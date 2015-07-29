package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"gopkg.in/alecthomas/kingpin.v2"
)

type Severity string

// Linter message severity levels.
const (
	Warning Severity = "warning"
	Error   Severity = "error"
)

type Linter string

func (l Linter) Command() string {
	s := lintersFlag[string(l)]
	return s[0:strings.Index(s, ":")]
}

func (l Linter) Pattern() string {
	s := lintersFlag[string(l)]
	return s[strings.Index(s, ":"):]
}

func (l Linter) InstallFrom() string {
	return installMap[string(l)]
}

func (l Linter) Severity() string {
	return linterSeverityFlag[string(l)]
}

func (l Linter) MessageOverride() string {
	return linterMessageOverrideFlag[string(l)]
}

type sortedIssues struct {
	issues []*Issue
	order  []string
}

func (s *sortedIssues) Len() int      { return len(s.issues) }
func (s *sortedIssues) Swap(i, j int) { s.issues[i], s.issues[j] = s.issues[j], s.issues[i] }
func (s *sortedIssues) Less(i, j int) bool {
	l, r := s.issues[i], s.issues[j]
	for _, key := range s.order {
		switch key {
		case "path":
			if l.path >= r.path {
				return false
			}
		case "line":
			if l.line >= r.line {
				return false
			}
		case "column":
			if l.col >= r.col {
				return false
			}
		case "severity":
			if l.severity >= r.severity {
				return false
			}
		case "message":
			if l.message >= r.message {
				return false
			}
		}
	}
	return true
}

var (
	predefinedPatterns = map[string]string{
		"PATH:LINE:COL:MESSAGE": `^(?P<path>[^\s][^:]+?\.go):(?P<line>\d+):(?P<col>\d+):\s*(?P<message>.*)$`,
		"PATH:LINE:MESSAGE":     `^(?P<path>[^\s][^:]+?\.go):(?P<line>\d+):\s*(?P<message>.*)$`,
	}
	lintersFlag = map[string]string{
		"golint":      "golint {path}:PATH:LINE:COL:MESSAGE",
		"vet":         "go tool vet {path}/*.go:PATH:LINE:MESSAGE",
		"gotype":      "gotype -e {tests=-a} {path}:PATH:LINE:COL:MESSAGE",
		"errcheck":    `errcheck {path}:^(?P<path>[^:]+):(?P<line>\d+):(?P<col>\d+)\t(?P<message>.*)$`,
		"varcheck":    `varcheck {path}:^(?:[^:]+: )?(?P<path>[^:]+):(?P<line>\d+):\s*(?P<message>\w+)$`,
		"structcheck": `structcheck {tests=-t} {path}:^(?:[^:]+: )?(?P<path>[^:]+):(?P<line>\d+):\s*(?P<message>[\w.]+)$`,
		"defercheck":  "defercheck {path}:PATH:LINE:MESSAGE",
		"deadcode":    `deadcode {path}:^deadcode: (?P<path>[^:]+):(?P<line>\d+):(?P<col>\d+):\s*(?P<message>.*)$`,
		"gocyclo":     `gocyclo -over {mincyclo} {path}:^(?P<cyclo>\d+)\s+\S+\s(?P<function>\S+)\s+(?P<path>[^:]+):(?P<line>\d+):(\d+)$`,
		"go-nyet":     `go-nyet {path}:PATH:LINE:COL:MESSAGE`,
		"ineffassign": `ineffassign -n {path}:^(?P<path>[^:]+):(?P<line>\d+):(?P<col>\d+)\s+(?P<message>.*)$`,
		"testify":     `go test:Location:\s+(?P<path>[^:]+):(?P<line>\d+)$\s+Error:\s+(?P<message>[^\n]+)`,
		"test":        `go test:^--- FAIL: .*$\s+(?P<path>[^:]+):(?P<line>\d+): (?P<message>.*)$`,
		"dupl":        `dupl -plumbing -threshold {duplthreshold} {path}/*.go:^(?P<path>[^\s][^:]+?\.go):(?P<line>\d+)-\d+:\s*(?P<message>.*)$`,
	}
	disabledLinters           = []string{"testify", "test"}
	enabledLinters            = []string{}
	linterMessageOverrideFlag = map[string]string{
		"errcheck":    "error return value not checked ({message})",
		"varcheck":    "unused global variable {message}",
		"structcheck": "unused struct field {message}",
		"gocyclo":     "cyclomatic complexity {cyclo} of function {function}() is high (> {mincyclo})",
		"ineffassign": `assignment to "{message}" is ineffective`,
	}
	linterSeverityFlag = map[string]string{
		"errcheck":    "warning",
		"golint":      "warning",
		"varcheck":    "warning",
		"structcheck": "warning",
		"deadcode":    "warning",
		"gocyclo":     "warning",
		"go-nyet":     "warning",
		"ineffassign": "warning",
		"dupl":        "warning",
	}
	installMap = map[string]string{
		"golint":      "github.com/golang/lint/golint",
		"gotype":      "golang.org/x/tools/cmd/gotype",
		"errcheck":    "github.com/alecthomas/errcheck",
		"defercheck":  "github.com/opennota/check/cmd/defercheck",
		"varcheck":    "github.com/opennota/check/cmd/varcheck",
		"structcheck": "github.com/opennota/check/cmd/structcheck",
		"deadcode":    "github.com/remyoudompheng/go-misc/deadcode",
		"gocyclo":     "github.com/alecthomas/gocyclo",
		"go-nyet":     "github.com/barakmich/go-nyet",
		"ineffassign": "github.com/gordonklaus/ineffassign",
		"dupl":        "github.com/mibk/dupl",
	}
	slowLinters = []string{"structcheck", "varcheck", "errcheck", "testify", "test"}
	sortKeys    = []string{"none", "path", "line", "column", "severity", "message"}

	pathsArg          = kingpin.Arg("path", "Directory to lint. Defaults to \".\". <path>/... will recurse.").Strings()
	fastFlag          = kingpin.Flag("fast", "Only run fast linters.").Bool()
	installFlag       = kingpin.Flag("install", "Attempt to install all known linters.").Short('i').Bool()
	updateFlag        = kingpin.Flag("update", "Pass -u to go tool when installing.").Short('u').Bool()
	debugFlag         = kingpin.Flag("debug", "Display messages for failed linters, etc.").Short('d').Bool()
	concurrencyFlag   = kingpin.Flag("concurrency", "Number of concurrent linters to run.").Default("16").Short('j').Int()
	excludeFlag       = kingpin.Flag("exclude", "Exclude messages matching these regular expressions.").Short('e').PlaceHolder("REGEXP").Strings()
	cycloFlag         = kingpin.Flag("cyclo-over", "Report functions with cyclomatic complexity over N (using gocyclo).").Default("10").Int()
	duplThresholdFlag = kingpin.Flag("dupl-threshold", "Minimum token sequence as a clone for dupl.").Default("50").Int()
	sortFlag          = kingpin.Flag("sort", fmt.Sprintf("Sort output by any of %s.", strings.Join(sortKeys, ", "))).Default("none").Enums(sortKeys...)
	testFlag          = kingpin.Flag("tests", "Include test files for linters that support this option").Short('t').Bool()
	deadlineFlag      = kingpin.Flag("deadline", "Cancel linters if they have not completed within this duration.").Default("5s").Duration()
	errorsFlag        = kingpin.Flag("errors", "Only show errors.").Bool()
)

func init() {
	kingpin.Flag("disable", fmt.Sprintf("List of linters to disable (%s).", strings.Join(disabledLinters, ","))).PlaceHolder("LINTER").Short('D').StringsVar(&disabledLinters)
	kingpin.Flag("enable", "Enable previously disabled linters.").PlaceHolder("LINTER").Short('E').StringsVar(&enabledLinters)
	kingpin.Flag("linter", "Specify a linter.").PlaceHolder("NAME:COMMAND:PATTERN").StringMapVar(&lintersFlag)
	kingpin.Flag("message-overrides", "Override message from linter. {message} will be expanded to the original message.").PlaceHolder("LINTER:MESSAGE").StringMapVar(&linterMessageOverrideFlag)
	kingpin.Flag("severity", "Map of linter severities.").PlaceHolder("LINTER:SEVERITY").StringMapVar(&linterSeverityFlag)
}

type Issue struct {
	linter   Linter
	severity Severity
	path     string
	line     int
	col      int
	message  string
}

func (i *Issue) String() string {
	col := ""
	if i.col != 0 {
		col = fmt.Sprintf("%d", i.col)
	}
	return fmt.Sprintf("%s:%d:%s:%s: %s (%s)", strings.TrimSpace(i.path), i.line, col, i.severity, strings.TrimSpace(i.message), i.linter)
}

func debug(format string, args ...interface{}) {
	if *debugFlag {
		fmt.Fprintf(os.Stderr, "DEBUG: "+format+"\n", args...)
	}
}

func formatLinters() string {
	w := bytes.NewBuffer(nil)
	for name := range lintersFlag {
		linter := Linter(name)
		fmt.Fprintf(w, "    %s (%s)\n        %s\n        %s\n", name, linter.InstallFrom(), linter.Command(), linter.Pattern())
	}
	return w.String()
}

func formatSeverity() string {
	w := bytes.NewBuffer(nil)
	for name, severity := range linterSeverityFlag {
		fmt.Fprintf(w, "    %s -> %s\n", name, severity)
	}
	return w.String()
}

func exArgs() (arg0 string, arg1 string) {
	if runtime.GOOS == "windows" {
		arg0 = "cmd"
		arg1 = "/C"
	} else {
		arg0 = "/bin/sh"
		arg1 = "-c"
	}
	return
}

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

func main() {
	kingpin.CommandLine.Help = fmt.Sprintf(`Aggregate and normalise the output of a whole bunch of Go linters.

Default linters:

%s

Severity override map (default is "error"):

%s
`, formatLinters(), formatSeverity())
	kingpin.Parse()
	var filter *regexp.Regexp
	if len(*excludeFlag) > 0 {
		filter = regexp.MustCompile(strings.Join(*excludeFlag, "|"))
	}

	if *fastFlag {
		disabledLinters = append(disabledLinters, slowLinters...)
	}

	if *installFlag {
		doInstall()
		return
	}

	runtime.GOMAXPROCS(*concurrencyFlag)

	disable := map[string]bool{}
	for _, linter := range disabledLinters {
		disable[linter] = true
	}
	for _, linter := range enabledLinters {
		delete(disable, linter)
	}

	start := time.Now()
	paths := expandPaths(*pathsArg)

	concurrency := make(chan bool, *concurrencyFlag)
	incomingIssues := make(chan *Issue, 1000000)
	processedIssues := maybeSortIssues(incomingIssues)
	wg := &sync.WaitGroup{}
	for name, description := range lintersFlag {
		if _, ok := disable[name]; ok {
			debug("linter %s disabled", name)
			continue
		}
		parts := strings.SplitN(description, ":", 2)
		command := parts[0]
		pattern := parts[1]

		// Recreated in each loop because it is mutated by executeLinter().
		vars := Vars{
			"duplthreshold": fmt.Sprintf("%d", *duplThresholdFlag),
			"mincyclo":      fmt.Sprintf("%d", *cycloFlag),
			"tests":         "",
		}
		if *testFlag {
			vars["tests"] = "-t"
		}
		for _, path := range paths {
			wg.Add(1)
			deadline := time.After(*deadlineFlag)
			state := &linterState{
				issues:   incomingIssues,
				name:     name,
				command:  command,
				pattern:  pattern,
				path:     path,
				vars:     vars.Copy(),
				filter:   filter,
				deadline: deadline,
			}
			go func() {
				concurrency <- true
				executeLinter(state)
				<-concurrency
				wg.Done()
			}()
		}
	}

	wg.Wait()
	close(incomingIssues)
	status := 0
	for issue := range processedIssues {
		if *errorsFlag && issue.severity != Error {
			continue
		}
		fmt.Println(issue.String())
		status = 1
	}
	elapsed := time.Now().Sub(start)
	debug("total elapsed time %s", elapsed)
	os.Exit(status)
}

func expandPaths(paths []string) []string {
	if len(paths) == 0 {
		paths = []string{"."}
	}
	dirs := map[string]bool{}
	for _, path := range paths {
		if strings.HasSuffix(path, "/...") {
			root := filepath.Dir(path)
			_ = filepath.Walk(root, func(p string, i os.FileInfo, err error) error {
				kingpin.FatalIfError(err, "invalid path '"+p+"'")

				base := filepath.Base(p)
				skip := strings.ContainsAny(base[0:1], "_.") && base != "." && base != ".."
				if i.IsDir() {
					if skip {
						return filepath.SkipDir
					}
				} else if !skip && strings.HasSuffix(p, ".go") {
					dirs[filepath.Clean(filepath.Dir(p))] = true
				}
				return nil
			})
		} else {
			dirs[filepath.Clean(path)] = true
		}
	}
	out := make([]string, 0, len(dirs))
	for d := range dirs {
		out = append(out, d)
	}
	return out
}

func doInstall() {
	for name, target := range installMap {
		cmd := "go get"
		if *debugFlag {
			cmd += " -v"
		}
		if *updateFlag {
			cmd += " -u"
		}
		cmd += " " + target
		fmt.Printf("Installing %s -> %s\n", name, cmd)
		arg0, arg1 := exArgs()
		c := exec.Command(arg0, arg1, cmd)
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr
		err := c.Run()
		kingpin.CommandLine.FatalIfError(err, "failed to install %s: %s", name, err)
	}
}

func maybeSortIssues(issues chan *Issue) chan *Issue {
	if reflect.DeepEqual([]string{"none"}, *sortFlag) {
		return issues
	}
	out := make(chan *Issue, 1000000)
	sorted := &sortedIssues{
		issues: []*Issue{},
		order:  *sortFlag,
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

type linterState struct {
	issues                       chan *Issue
	name, command, pattern, path string
	vars                         Vars
	filter                       *regexp.Regexp
	deadline                     <-chan time.Time
}

func (l *linterState) InterpolatedCommand() string {
	l.vars["path"] = l.path
	return l.vars.Replace(l.command)
}

func (l *linterState) Match() *regexp.Regexp {
	re, err := regexp.Compile("(?m:" + l.pattern + ")")
	kingpin.FatalIfError(err, "invalid pattern for '"+l.command+"'")
	return re
}

func executeLinter(state *linterState) {
	debug("linting with %s: %s (on %s)", state.name, state.command, state.path)

	start := time.Now()
	if p, ok := predefinedPatterns[state.pattern]; ok {
		state.pattern = p
	}

	command := state.InterpolatedCommand()
	debug("executing %s", command)
	arg0, arg1 := exArgs()
	buf := bytes.NewBuffer(nil)
	cmd := exec.Command(arg0, arg1, command)
	cmd.Stdout = buf
	cmd.Stderr = buf
	err := cmd.Start()
	if err == nil {
		done := make(chan bool)
		go func() {
			err = cmd.Wait()
			done <- true
		}()

		// Wait for process to complete or deadline to expire.
		select {
		case <-done:

		case <-state.deadline:
			debug("warning: deadline exceeded by linter %s", state.name)
			_ = cmd.Process.Kill()
			return
		}
	}

	if err != nil {
		if _, ok := err.(*exec.ExitError); !ok {
			debug("warning: %s failed: %s", command, err)
			return
		}
		debug("warning: %s returned %s", command, err)
	}

	processOutput(state, buf.Bytes())
	elapsed := time.Now().Sub(start)
	debug("%s linter took %s", state.name, elapsed)
}

func processOutput(state *linterState, out []byte) {
	re := state.Match()
	all := re.FindAllSubmatchIndex(out, -1)
	debug("%s hits %d: %s", state.name, len(all), state.pattern)
	for _, indices := range all {
		group := [][]byte{}
		for i := 0; i < len(indices); i += 2 {
			fragment := out[indices[i]:indices[i+1]]
			group = append(group, fragment)
		}

		issue := &Issue{}
		issue.linter = Linter(state.name)
		for i, name := range re.SubexpNames() {
			part := string(group[i])
			if name != "" {
				state.vars[name] = part
			}
			switch name {
			case "path":
				issue.path = part

			case "line":
				n, err := strconv.ParseInt(part, 10, 32)
				kingpin.FatalIfError(err, "line matched invalid integer")
				issue.line = int(n)

			case "col":
				n, err := strconv.ParseInt(part, 10, 32)
				kingpin.FatalIfError(err, "col matched invalid integer")
				issue.col = int(n)

			case "message":
				issue.message = part

			case "":
			}
		}
		if m, ok := linterMessageOverrideFlag[state.name]; ok {
			issue.message = state.vars.Replace(m)
		}
		if sev, ok := linterSeverityFlag[state.name]; ok {
			issue.severity = Severity(sev)
		} else {
			issue.severity = "error"
		}
		if state.filter != nil && state.filter.MatchString(issue.String()) {
			continue
		}
		state.issues <- issue
	}
	return
}
