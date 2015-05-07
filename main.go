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

	"gopkg.in/alecthomas/kingpin.v2-unstable"
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
		"PATH:LINE:COL:MESSAGE": `^(?P<path>[^:]+?\.go):(?P<line>\d+):(?P<col>\d+):\s*(?P<message>.*)$`,
		"PATH:LINE:MESSAGE":     `^(?P<path>[^:]+?\.go):(?P<line>\d+):\s*(?P<message>.*)$`,
	}
	lintersFlag = map[string]string{
		// main.go:8:10: should omit type map[string]string from declaration of var linters; it will be inferred from the right-hand side
		"golint": "golint {path}:PATH:LINE:COL:MESSAGE",
		// test/stutter.go:19: missing argument for Printf("%d"): format reads arg 1, have only 0 args
		"vet":         "go vet {path}:PATH:LINE:MESSAGE",
		"gotype":      "gotype {tests=-a} {path}:PATH:LINE:COL:MESSAGE",
		"errcheck":    `errcheck {path}:^(?P<path>[^:]+):(?P<line>\d+):(?P<col>\d+)\t(?P<message>.*)$`,
		"varcheck":    `varcheck {path}:^(?:[^:]+: )?(?P<path>[^:]+):(?P<line>\d+):(?P<col>\d+):\s*(?P<message>\w+)$`,
		"structcheck": `structcheck {tests=-t} {path}:^(?:[^:]+: )?(?P<path>[^:]+):(?P<line>\d+):\s*(?P<message>[\w.]+)$`,
		"defercheck":  "defercheck {path}:PATH:LINE:MESSAGE",
		"deadcode":    `deadcode {path}:deadcode: (?P<path>[^:]+):(?P<line>\d+):(?P<col>\d+):\s*(?P<message>.*)`,
		"gocyclo":     `gocyclo -over {mincyclo} {path}:^(?P<cyclo>\d+)\s+\S+\s(?P<function>\S+)\s+(?P<path>[^:]+):(?P<line>\d+):(?P<col>\d+)`,
		"go-nyet":     `go-nyet {path}:PATH:LINE:COL:MESSAGE`,
	}
	linterMessageOverrideFlag = map[string]string{
		"errcheck":    "error return value not checked ({message})",
		"varcheck":    "unused global variable {message}",
		"structcheck": "unused struct field {message}",
		"gocyclo":     "cyclomatic complexity {cyclo} of function {function}() is high (> {mincyclo})",
	}
	linterSeverityFlag = map[string]string{
		"errcheck":    "warning",
		"golint":      "warning",
		"varcheck":    "warning",
		"structcheck": "warning",
		"deadcode":    "warning",
		"gocyclo":     "warning",
		"go-nyet":     "warning",
	}
	installMap = map[string]string{
		"golint":      "github.com/golang/lint/golint",
		"gotype":      "golang.org/x/tools/cmd/gotype",
		"errcheck":    "github.com/kisielk/errcheck",
		"defercheck":  "github.com/opennota/check/cmd/defercheck",
		"varcheck":    "github.com/opennota/check/cmd/varcheck",
		"structcheck": "github.com/opennota/check/cmd/structcheck",
		"deadcode":    "github.com/remyoudompheng/go-misc/deadcode",
		"gocyclo":     "github.com/fzipp/gocyclo",
		"go-nyet":     "github.com/barakmich/go-nyet",
	}
	slowLinters = []string{"structcheck", "varcheck", "errcheck"}
	sortKeys    = []string{"none", "path", "line", "column", "severity", "message"}

	pathsArg           = kingpin.Arg("path", "Directory to lint.").Strings()
	fastFlag           = kingpin.Flag("fast", "Only run fast linters.").Bool()
	installFlag        = kingpin.Flag("install", "Attempt to install all known linters.").Short('i').Bool()
	updateFlag         = kingpin.Flag("update", "Pass -u to go tool when installing.").Short('u').Bool()
	disableLintersFlag = kingpin.Flag("disable", "List of linters to disable.").PlaceHolder("LINTER").Short('D').Strings()
	debugFlag          = kingpin.Flag("debug", "Display messages for failed linters, etc.").Short('d').Bool()
	concurrencyFlag    = kingpin.Flag("concurrency", "Number of concurrent linters to run.").Default("16").Short('j').Int()
	excludeFlag        = kingpin.Flag("exclude", "Exclude messages matching this regular expression.").PlaceHolder("REGEXP").String()
	cycloFlag          = kingpin.Flag("cyclo-over", "Report functions with cyclomatic complexity over N (using gocyclo).").Default("10").Int()
	sortFlag           = kingpin.Flag("sort", fmt.Sprintf("Sort output by any of %s.", strings.Join(sortKeys, ", "))).Default("none").Enums(sortKeys...)
	testFlag           = kingpin.Flag("tests", "Include test files for linters that support this option").Short('t').Bool()
	deadlineFlag       = kingpin.Flag("deadline", "Cancel linters if they have not completed within this duration.").Default("5s").Duration()
	errorsFlag         = kingpin.Flag("errors", "Only show errors.").Bool()
)

func init() {
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
	return fmt.Sprintf("%s:%d:%s:%s: %s (%s)", i.path, i.line, col, i.severity, i.message, i.linter)
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
	if *excludeFlag != "" {
		filter = regexp.MustCompile(*excludeFlag)
	}

	if *fastFlag {
		*disableLintersFlag = append(*disableLintersFlag, slowLinters...)
	}

	if *installFlag {
		doInstall()
		return
	}

	runtime.GOMAXPROCS(*concurrencyFlag)

	disable := map[string]bool{}
	for _, linter := range *disableLintersFlag {
		disable[linter] = true
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
			"mincyclo": fmt.Sprintf("%d", *cycloFlag),
			"tests":    "",
		}
		if *testFlag {
			vars["tests"] = "-t"
		}
		for _, path := range paths {
			wg.Add(1)
			go func(path, name, command, pattern string) {
				concurrency <- true
				state := &linterState{
					issues:   incomingIssues,
					name:     name,
					command:  command,
					pattern:  pattern,
					path:     path,
					vars:     vars,
					filter:   filter,
					deadline: time.After(*deadlineFlag),
				}
				executeLinter(state)
				<-concurrency
				wg.Done()
			}(path, name, command, pattern)
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
	out := []string{}
	for _, path := range paths {
		if strings.HasSuffix(path, "/...") {
			root := filepath.Dir(path)
			_ = filepath.Walk(root, func(p string, i os.FileInfo, err error) error {
				kingpin.FatalIfError(err, "invalid path '"+p+"'")

				if i.IsDir() {
					base := filepath.Base(p)
					if strings.HasPrefix(base, ".") && base != "." && base != ".." {
						return filepath.SkipDir
					}
					out = append(out, filepath.Clean(p))
				}
				return nil
			})
		} else {
			out = append(out, filepath.Clean(path))
		}
	}

	// Deduplicate paths.
	sort.Strings(out)
	clean := []string{}
	last := ""
	for _, path := range out {
		if path != last {
			clean = append(clean, path)
		}
		last = path
	}
	return clean
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
		if err != nil {
			kingpin.CommandLine.Errorf(os.Stderr, "failed to install %s: %s", name, err)
		}
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
	re, err := regexp.Compile(l.pattern)
	kingpin.FatalIfError(err, "invalid pattern for '"+l.command+"'")
	return re
}

func executeLinter(state *linterState) {
	debug("linting with %s: %s", state.name, state.command)

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
	count := 0
	re := state.Match()
	for _, line := range bytes.Split(out, []byte("\n")) {
		groups := re.FindAllSubmatch(line, -1)
		if groups == nil {
			debug("%s (didn't match): '%s'", state.name, line)
			continue
		}
		issue := &Issue{}
		issue.linter = Linter(state.name)
		for i, name := range re.SubexpNames() {
			part := string(groups[0][i])
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
		count++
		state.issues <- issue
	}
	return
}
