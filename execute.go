package main

import (
	"bytes"
	"fmt"
	"os/exec"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/shlex"
	"gopkg.in/alecthomas/kingpin.v3-unstable"
)

type Vars map[string]string

func (v Vars) Copy() Vars {
	out := Vars{}
	for k, v := range v {
		out[k] = v
	}
	return out
}

func (v Vars) Replace(s string) string {
	for k, v := range v {
		prefix := regexp.MustCompile(fmt.Sprintf("{%s=([^}]*)}", k))
		if v != "" {
			s = prefix.ReplaceAllString(s, "$1")
		} else {
			s = prefix.ReplaceAllString(s, "")
		}
		s = strings.Replace(s, fmt.Sprintf("{%s}", k), v, -1)
	}
	return s
}

// Severity of linter message.
type Severity string

// Linter message severity levels.
const ( // nolint: deadcode
	Error   Severity = "error"
	Warning Severity = "warning"
)

type Issue struct {
	Linter   *Linter  `json:"linter"`
	Severity Severity `json:"severity"`
	Path     string   `json:"path"`
	Line     int      `json:"line"`
	Col      int      `json:"col"`
	Message  string   `json:"message"`
}

func (i *Issue) String() string {
	buf := new(bytes.Buffer)
	err := formatTemplate.Execute(buf, i)
	kingpin.FatalIfError(err, "Invalid output format")
	return buf.String()
}

type linterState struct {
	*Linter
	path     string
	issues   chan *Issue
	vars     Vars
	exclude  *regexp.Regexp
	include  *regexp.Regexp
	deadline <-chan time.Time
}

func (l *linterState) InterpolatedCommand() string {
	vars := l.vars.Copy()
	if l.ShouldChdir() {
		vars["path"] = "."
	} else {
		vars["path"] = l.path
	}
	return vars.Replace(l.Command)
}

func (l *linterState) ShouldChdir() bool {
	return config.Vendor || !strings.HasSuffix(l.path, "/...") || !strings.Contains(l.Command, "{path}")
}

func (l *linterState) fixPath(path string) string {
	lpath := strings.TrimSuffix(l.path, "...")
	labspath, _ := filepath.Abs(lpath)

	if !l.ShouldChdir() {
		path = strings.TrimPrefix(path, lpath)
	}

	if !filepath.IsAbs(path) {
		path, _ = filepath.Abs(filepath.Join(labspath, path))
	}
	if strings.HasPrefix(path, labspath) {
		return filepath.Join(lpath, strings.TrimPrefix(path, labspath))
	}
	return path
}

func runLinters(linters map[string]*Linter, paths, ellipsisPaths []string, concurrency int, exclude *regexp.Regexp, include *regexp.Regexp) (chan *Issue, chan error) {
	errch := make(chan error, len(linters)*(len(paths)+len(ellipsisPaths)))
	concurrencych := make(chan bool, concurrency)
	incomingIssues := make(chan *Issue, 1000000)
	directives := newDirectiveParser()
	processedIssues := filterIssuesViaDirectives(directives, maybeSortIssues(maybeAggregateIssues(incomingIssues)))
	wg := &sync.WaitGroup{}
	for _, linter := range linters {
		// Recreated in each loop because it is mutated by executeLinter().
		vars := Vars{
			"duplthreshold":    fmt.Sprintf("%d", config.DuplThreshold),
			"mincyclo":         fmt.Sprintf("%d", config.Cyclo),
			"maxlinelength":    fmt.Sprintf("%d", config.LineLength),
			"min_confidence":   fmt.Sprintf("%f", config.MinConfidence),
			"min_occurrences":  fmt.Sprintf("%d", config.MinOccurrences),
			"min_const_length": fmt.Sprintf("%d", config.MinConstLength),
			"tests":            "",
		}
		if config.Test {
			vars["tests"] = "-t"
		}
		linterPaths := paths
		// Most linters don't exclude vendor paths when recursing, so we don't use ... paths.
		if acceptsEllipsis[linter.Name] && !config.Vendor && len(ellipsisPaths) > 0 {
			linterPaths = ellipsisPaths
		}
		for _, path := range linterPaths {
			wg.Add(1)
			deadline := time.After(config.Deadline)
			state := &linterState{
				Linter:   linter,
				issues:   incomingIssues,
				path:     path,
				vars:     vars.Copy(),
				exclude:  exclude,
				include:  include,
				deadline: deadline,
			}
			go func() {
				concurrencych <- true
				err := executeLinter(state)
				if err != nil {
					errch <- err
				}
				<-concurrencych
				wg.Done()
			}()
		}
	}

	go func() {
		wg.Wait()
		close(incomingIssues)
		close(errch)
	}()
	return processedIssues, errch
}

func executeLinter(state *linterState) error {
	debug("linting with %s: %s (on %s)", state.Name, state.Command, state.path)

	start := time.Now()
	command := state.InterpolatedCommand()
	exe, args, err := parseCommand(state.path, command)
	if err != nil {
		return err
	}
	debug("executing %s %q", exe, args)
	buf := bytes.NewBuffer(nil)
	cmd := exec.Command(exe, args...) // nolint: gas
	if state.ShouldChdir() {
		cmd.Dir = state.path
	}
	cmd.Stdout = buf
	cmd.Stderr = buf
	err = cmd.Start()
	if err != nil {
		return fmt.Errorf("failed to execute linter %s: %s", command, err)
	}

	done := make(chan bool)
	go func() {
		err = cmd.Wait()
		done <- true
	}()

	// Wait for process to complete or deadline to expire.
	select {
	case <-done:

	case <-state.deadline:
		err = fmt.Errorf("deadline exceeded by linter %s on %s (try increasing --deadline)",
			state.Name, state.path)
		kerr := cmd.Process.Kill()
		if kerr != nil {
			warning("failed to kill %s: %s", state.Name, kerr)
		}
		return err
	}

	if err != nil {
		debug("warning: %s returned %s", command, err)
	}

	processOutput(state, buf.Bytes())
	elapsed := time.Since(start)
	debug("%s linter took %s", state.Name, elapsed)
	return nil
}

func parseCommand(dir, command string) (string, []string, error) {
	args, err := shlex.Split(command)
	if err != nil {
		return "", nil, err
	}
	if len(args) == 0 {
		return "", nil, fmt.Errorf("invalid command %q", command)
	}
	exe, err := exec.LookPath(args[0])
	if err != nil {
		return "", nil, err
	}
	out := []string{}
	for _, arg := range args[1:] {
		if strings.Contains(arg, "*") {
			pattern := filepath.Join(dir, arg)
			globbed, err := filepath.Glob(pattern)
			if err != nil {
				return "", nil, err
			}
			for i, g := range globbed {
				if strings.HasPrefix(g, dir+string(filepath.Separator)) {
					globbed[i] = g[len(dir)+1:]
				}
			}
			out = append(out, globbed...)
		} else {
			out = append(out, arg)
		}
	}
	return exe, out, nil
}

// nolint: gocyclo
func processOutput(state *linterState, out []byte) {
	re := state.regex
	all := re.FindAllSubmatchIndex(out, -1)
	debug("%s hits %d: %s", state.Name, len(all), state.Pattern)
	for _, indices := range all {
		group := [][]byte{}
		for i := 0; i < len(indices); i += 2 {
			var fragment []byte
			if indices[i] != -1 {
				fragment = out[indices[i]:indices[i+1]]
			}
			group = append(group, fragment)
		}

		issue := &Issue{Line: 1}
		issue.Linter = LinterFromName(state.Name)
		for i, name := range re.SubexpNames() {
			if group[i] == nil {
				continue
			}
			part := string(group[i])
			if name != "" {
				state.vars[name] = part
			}
			switch name {
			case "path":
				issue.Path = state.fixPath(part)

			case "line":
				n, err := strconv.ParseInt(part, 10, 32)
				kingpin.FatalIfError(err, "line matched invalid integer")
				issue.Line = int(n)

			case "col":
				n, err := strconv.ParseInt(part, 10, 32)
				kingpin.FatalIfError(err, "col matched invalid integer")
				issue.Col = int(n)

			case "message":
				issue.Message = part

			case "":
			}
		}
		if m, ok := config.MessageOverride[state.Name]; ok {
			issue.Message = state.vars.Replace(m)
		}
		if sev, ok := config.Severity[state.Name]; ok {
			issue.Severity = Severity(sev)
		} else {
			issue.Severity = "warning"
		}
		if state.exclude != nil && state.exclude.MatchString(issue.String()) {
			continue
		}
		if state.include != nil && !state.include.MatchString(issue.String()) {
			continue
		}
		state.issues <- issue
	}
	return
}

func maybeAggregateIssues(issues chan *Issue) chan *Issue {
	if !config.Aggregate {
		return issues
	}
	return aggregateIssues(issues)
}

type sortedIssues struct {
	issues []*Issue
	order  []string
}

func (s *sortedIssues) Len() int      { return len(s.issues) }
func (s *sortedIssues) Swap(i, j int) { s.issues[i], s.issues[j] = s.issues[j], s.issues[i] }

// nolint: gocyclo
func (s *sortedIssues) Less(i, j int) bool {
	l, r := s.issues[i], s.issues[j]
	for _, key := range s.order {
		switch key {
		case "path":
			if l.Path > r.Path {
				return false
			}
		case "line":
			if l.Line > r.Line {
				return false
			}
		case "column":
			if l.Col > r.Col {
				return false
			}
		case "severity":
			if l.Severity > r.Severity {
				return false
			}
		case "message":
			if l.Message > r.Message {
				return false
			}
		case "linter":
			if l.Linter.Name > r.Linter.Name {
				return false
			}
		}
	}
	return true
}

func maybeSortIssues(issues chan *Issue) chan *Issue {
	if reflect.DeepEqual([]string{"none"}, config.Sort) {
		return issues
	}
	out := make(chan *Issue, 1000000)
	sorted := &sortedIssues{
		issues: []*Issue{},
		order:  config.Sort,
	}
	go func() {
		for issue := range issues {
			sorted.issues = append(sorted.issues, issue)
		}
		sort.Sort(sorted)
		for _, issue := range sorted.issues {
			out <- issue
		}
		close(out)
	}()
	return out
}
