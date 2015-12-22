package main

import (
	"bytes"
	"encoding/json"
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

	"github.com/google/shlex"
	"gopkg.in/alecthomas/kingpin.v2"
)

type Severity string

// Linter message severity levels.
const (
	Warning Severity = "warning"
	Error   Severity = "error"
)

type Linter struct {
	Name             string
	Command          string
	Pattern          string
	InstallFrom      string
	SeverityOverride Severity
	MessageOverride  string
}

func (l *Linter) MarshalJSON() ([]byte, error) {
	return json.Marshal(l.Name)
}

func (l *Linter) String() string {
	return l.Name
}

func LinterFromName(name string) *Linter {
	s := lintersFlag[name]
	return &Linter{
		Name:             name,
		Command:          s[0:strings.Index(s, ":")],
		Pattern:          s[strings.Index(s, ":"):],
		InstallFrom:      installMap[name],
		SeverityOverride: Severity(linterSeverityFlag[name]),
		MessageOverride:  linterMessageOverrideFlag[name],
	}
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
			if l.Path >= r.Path {
				return false
			}
		case "line":
			if l.Line >= r.Line {
				return false
			}
		case "column":
			if l.Col >= r.Col {
				return false
			}
		case "severity":
			if l.Severity >= r.Severity {
				return false
			}
		case "message":
			if l.Message >= r.Message {
				return false
			}
		case "linter":
			if l.Linter.Name >= r.Linter.Name {
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
		"golint":      "golint -min_confidence {min_confidence} .:PATH:LINE:COL:MESSAGE",
		"vet":         "go tool vet ./*.go:PATH:LINE:MESSAGE",
		"vetshadow":   "go tool vet --shadow ./*.go:PATH:LINE:MESSAGE",
		"gofmt":       `gofmt -l -s ./*.go:^(?P<path>[^\n]+)$`,
		"gotype":      "gotype -e {tests=-a} .:PATH:LINE:COL:MESSAGE",
		"goimports":   `goimports -l ./*.go:^(?P<path>[^\n]+)$`,
		"errcheck":    `errcheck -abspath .:^(?P<path>[^:]+):(?P<line>\d+):(?P<col>\d+)\t(?P<message>.*)$`,
		"varcheck":    `varcheck .:^(?:[^:]+: )?(?P<path>[^:]+):(?P<line>\d+):(?P<col>\d+):\s*(?P<message>\w+)$`,
		"structcheck": `structcheck {tests=-t} .:^(?:[^:]+: )?(?P<path>[^:]+):(?P<line>\d+):(?P<col>\d+):\s*(?P<message>.+)$`,
		"aligncheck":  `aligncheck .:^(?:[^:]+: )?(?P<path>[^:]+):(?P<line>\d+):(?P<col>\d+):\s*(?P<message>.+)$`,
		"deadcode":    `deadcode .:^deadcode: (?P<path>[^:]+):(?P<line>\d+):(?P<col>\d+):\s*(?P<message>.*)$`,
		"gocyclo":     `gocyclo -over {mincyclo} .:^(?P<cyclo>\d+)\s+\S+\s(?P<function>\S+)\s+(?P<path>[^:]+):(?P<line>\d+):(\d+)$`,
		"ineffassign": `ineffassign -n .:PATH:LINE:COL:MESSAGE`,
		"testify":     `go test:Location:\s+(?P<path>[^:]+):(?P<line>\d+)$\s+Error:\s+(?P<message>[^\n]+)`,
		"test":        `go test:^--- FAIL: .*$\s+(?P<path>[^:]+):(?P<line>\d+): (?P<message>.*)$`,
		"dupl":        `dupl -plumbing -threshold {duplthreshold} ./*.go:^(?P<path>[^\s][^:]+?\.go):(?P<line>\d+)-\d+:\s*(?P<message>.*)$`,
	}
	disabledLinters           = []string{"testify", "test", "gofmt", "goimports"}
	enabledLinters            = []string{}
	linterMessageOverrideFlag = map[string]string{
		"errcheck":    "error return value not checked ({message})",
		"varcheck":    "unused global variable {message}",
		"structcheck": "unused struct field {message}",
		"gocyclo":     "cyclomatic complexity {cyclo} of function {function}() is high (> {mincyclo})",
		"gofmt":       "file is not gofmted",
		"goimports":   "file is not goimported",
	}
	linterSeverityFlag = map[string]string{
		"errcheck":    "warning",
		"golint":      "warning",
		"varcheck":    "warning",
		"structcheck": "warning",
		"aligncheck":  "warning",
		"deadcode":    "warning",
		"gocyclo":     "warning",
		"ineffassign": "warning",
		"dupl":        "warning",
		"gofmt":       "warning",
		"goimports":   "warning",
		"vetshadow":   "warning",
	}
	installMap = map[string]string{
		"gometalinter": "github.com/alecthomas/gometalinter",
		"golint":       "github.com/golang/lint/golint",
		"gotype":       "golang.org/x/tools/cmd/gotype",
		"goimports":    "golang.org/x/tools/cmd/goimports",
		"errcheck":     "github.com/kisielk/errcheck",
		"varcheck":     "github.com/opennota/check/cmd/varcheck",
		"structcheck":  "github.com/opennota/check/cmd/structcheck",
		"aligncheck":   "github.com/opennota/check/cmd/aligncheck",
		"deadcode":     "github.com/remyoudompheng/go-misc/deadcode",
		"gocyclo":      "github.com/alecthomas/gocyclo",
		"ineffassign":  "github.com/gordonklaus/ineffassign",
		"dupl":         "github.com/mibk/dupl",
	}
	slowLinters = []string{"structcheck", "varcheck", "errcheck", "aligncheck", "testify", "test"}
	sortKeys    = []string{"none", "path", "line", "column", "severity", "message", "linter"}

	pathsArg          = kingpin.Arg("path", "Directory to lint. Defaults to \".\". <path>/... will recurse.").Strings()
	fastFlag          = kingpin.Flag("fast", "Only run fast linters.").Bool()
	installFlag       = kingpin.Flag("install", "Attempt to install all known linters.").Short('i').Bool()
	updateFlag        = kingpin.Flag("update", "Pass -u to go tool when installing.").Short('u').Bool()
	forceFlag         = kingpin.Flag("force", "Pass -f to go tool when installing.").Short('f').Bool()
	debugFlag         = kingpin.Flag("debug", "Display messages for failed linters, etc.").Short('d').Bool()
	concurrencyFlag   = kingpin.Flag("concurrency", "Number of concurrent linters to run.").Default("16").Short('j').Int()
	excludeFlag       = kingpin.Flag("exclude", "Exclude messages matching these regular expressions.").Short('e').PlaceHolder("REGEXP").Strings()
	includeFlag       = kingpin.Flag("include", "Include messages matching these regular expressions.").Short('I').PlaceHolder("REGEXP").Strings()
	skipFlag          = kingpin.Flag("skip", "Skip directories with this name when expanding '...'.").Short('s').PlaceHolder("DIR...").Strings()
	vendorFlag        = kingpin.Flag("vendor", "Enable vendoring support (skips 'vendor' directories and sets GO15VENDOREXPERIMENT=1).").Bool()
	cycloFlag         = kingpin.Flag("cyclo-over", "Report functions with cyclomatic complexity over N (using gocyclo).").Default("10").Int()
	minConfidence     = kingpin.Flag("min-confidence", "Minimum confidence interval to pass to golint").Default(".80").Float()
	duplThresholdFlag = kingpin.Flag("dupl-threshold", "Minimum token sequence as a clone for dupl.").Default("50").Int()
	sortFlag          = kingpin.Flag("sort", fmt.Sprintf("Sort output by any of %s.", strings.Join(sortKeys, ", "))).Default("none").Enums(sortKeys...)
	testFlag          = kingpin.Flag("tests", "Include test files for linters that support this option").Short('t').Bool()
	deadlineFlag      = kingpin.Flag("deadline", "Cancel linters if they have not completed within this duration.").Default("5s").Duration()
	errorsFlag        = kingpin.Flag("errors", "Only show errors.").Bool()
	jsonFlag          = kingpin.Flag("json", "Generate structured JSON rather than standard line-based output.").Bool()
)

func disableAllLinters(*kingpin.ParseContext) error {
	disabledLinters = []string{}
	for linter := range lintersFlag {
		disabledLinters = append(disabledLinters, linter)
	}
	return nil
}

func init() {
	kingpin.Flag("disable", fmt.Sprintf("List of linters to disable (%s).", strings.Join(disabledLinters, ","))).PlaceHolder("LINTER").Short('D').StringsVar(&disabledLinters)
	kingpin.Flag("enable", "Enable previously disabled linters.").PlaceHolder("LINTER").Short('E').StringsVar(&enabledLinters)
	kingpin.Flag("linter", "Specify a linter.").PlaceHolder("NAME:COMMAND:PATTERN").StringMapVar(&lintersFlag)
	kingpin.Flag("message-overrides", "Override message from linter. {message} will be expanded to the original message.").PlaceHolder("LINTER:MESSAGE").StringMapVar(&linterMessageOverrideFlag)
	kingpin.Flag("severity", "Map of linter severities.").PlaceHolder("LINTER:SEVERITY").StringMapVar(&linterSeverityFlag)
	kingpin.Flag("disable-all", "Disable all linters.").Action(disableAllLinters).Bool()
}

type Issue struct {
	Linter   *Linter  `json:"linter"`
	Severity Severity `json:"severity"`
	Path     string   `json:"path"`
	Line     int      `json:"line"`
	Col      int      `json:"col"`
	Message  string   `json:"message"`
}

func (i *Issue) String() string {
	col := ""
	if i.Col != 0 {
		col = fmt.Sprintf("%d", i.Col)
	}
	return fmt.Sprintf("%s:%d:%s:%s: %s (%s)", strings.TrimSpace(i.Path), i.Line, col, i.Severity, strings.TrimSpace(i.Message), i.Linter)
}

func debug(format string, args ...interface{}) {
	if *debugFlag {
		fmt.Fprintf(os.Stderr, "DEBUG: "+format+"\n", args...)
	}
}

func warning(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "WARNING: "+format+"\n", args...)
}

func formatLinters() string {
	w := bytes.NewBuffer(nil)
	for name := range lintersFlag {
		linter := LinterFromName(name)
		install := "(" + linter.InstallFrom + ")"
		if install == "()" {
			install = ""
		}
		fmt.Fprintf(w, "  %s  %s\n        %s\n        %s\n", name, install, linter.Command, linter.Pattern)
	}
	return w.String()
}

func formatSeverity() string {
	w := bytes.NewBuffer(nil)
	for name, severity := range linterSeverityFlag {
		fmt.Fprintf(w, "  %s -> %s\n", name, severity)
	}
	return w.String()
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
	// Linters are by their very nature, short lived, so use sbrk for
	// allocations rather than GC. Reduced (user) linting time on kingpin from
	// 0.97s to 0.64s.
	_ = os.Setenv("GODEBUG", "sbrk=1")
	kingpin.CommandLine.Help = fmt.Sprintf(`Aggregate and normalise the output of a whole bunch of Go linters.

Default linters:

%s

Severity override map (default is "error"):

%s
`, formatLinters(), formatSeverity())
	kingpin.Parse()
	// Default to skipping "vendor" directory if GO15VENDOREXPERIMENT=1 is enabled.
	// TODO(alec): This will probably need to be enabled by default at a later time.
	if os.Getenv("GO15VENDOREXPERIMENT") == "1" || *vendorFlag {
		os.Setenv("GO15VENDOREXPERIMENT", "1")
		*skipFlag = append(*skipFlag, "vendor")
	}
	var exclude *regexp.Regexp
	if len(*excludeFlag) > 0 {
		exclude = regexp.MustCompile(strings.Join(*excludeFlag, "|"))
	}

	var include *regexp.Regexp
	if len(*includeFlag) > 0 {
		include = regexp.MustCompile(strings.Join(*includeFlag, "|"))
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
	paths := expandPaths(*pathsArg, *skipFlag)

	status := 0
	issues, errch := runLinters(lintersFlag, disable, paths, *concurrencyFlag, exclude, include)
	if *jsonFlag {
		status |= outputToJSON(issues)
	} else {
		status |= outputToConsole(issues)
	}
	for err := range errch {
		warning("%s", err)
		status |= 2
	}
	elapsed := time.Now().Sub(start)
	debug("total elapsed time %s", elapsed)
	os.Exit(status)
}

func outputToConsole(issues chan *Issue) int {
	status := 0
	for issue := range issues {
		if *errorsFlag && issue.Severity != Error {
			continue
		}
		fmt.Println(issue.String())
		status = 1
	}
	return status
}

func outputToJSON(issues chan *Issue) int {
	fmt.Println("[")
	status := 0
	for issue := range issues {
		if status != 0 {
			fmt.Printf(",\n")
		}
		if *errorsFlag && issue.Severity != Error {
			continue
		}
		d, err := json.Marshal(issue)
		kingpin.FatalIfError(err, "")
		fmt.Printf("  %s", d)
		status = 1
	}
	fmt.Printf("\n]\n")
	return status
}

func runLinters(linters map[string]string, disable map[string]bool, paths []string, concurrency int, exclude *regexp.Regexp, include *regexp.Regexp) (chan *Issue, chan error) {
	errch := make(chan error, len(linters)*len(paths))
	concurrencych := make(chan bool, *concurrencyFlag)
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
			"duplthreshold":  fmt.Sprintf("%d", *duplThresholdFlag),
			"mincyclo":       fmt.Sprintf("%d", *cycloFlag),
			"min_confidence": fmt.Sprintf("%f", *minConfidence),
			"tests":          "",
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

func expandPaths(paths, skip []string) []string {
	if len(paths) == 0 {
		paths = []string{"."}
	}
	skipMap := map[string]bool{}
	for _, name := range skip {
		skipMap[name] = true
	}
	dirs := map[string]bool{}
	for _, path := range paths {
		if strings.HasSuffix(path, "/...") {
			root := filepath.Dir(path)
			_ = filepath.Walk(root, func(p string, i os.FileInfo, err error) error {
				if err != nil {
					warning("invalid path %q: %s", p, err)
					return err
				}

				base := filepath.Base(p)
				skip := skipMap[base] || skipMap[p] || (strings.ContainsAny(base[0:1], "_.") && base != "." && base != "..")
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
	sort.Strings(out)
	for _, d := range out {
		debug("linting path %s", d)
	}
	return out
}

func doInstall() {
	cmd := "go get"
	if *debugFlag {
		cmd += " -v"
	}
	if *updateFlag {
		cmd += " -u"
	}
	if *forceFlag {
		cmd += " -f"
	}

	names := make([]string, 0, len(installMap))
	targets := make([]string, 0, len(installMap))
	for name, target := range installMap {
		names = append(names, name)
		targets = append(targets, target)
	}
	namesStr := strings.Join(names, "\n  ")
	targetsStr := strings.Join(targets, " ")
	cmd += " " + targetsStr
	fmt.Printf("Installing:\n  %s\n->\n  %s\n", namesStr, cmd)

	exe, args, err := parseCommand(".", cmd)
	kingpin.FatalIfError(err, "failed to parse command line: %s", err)
	c := exec.Command(exe, args...)
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	err = c.Run()
	kingpin.FatalIfError(err, "failed to install %s: %s", namesStr, err)
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
	exclude                      *regexp.Regexp
	include                      *regexp.Regexp
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
				if strings.HasPrefix(g, dir+"/") {
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

func executeLinter(state *linterState) error {
	debug("linting with %s: %s (on %s)", state.name, state.command, state.path)

	start := time.Now()
	if p, ok := predefinedPatterns[state.pattern]; ok {
		state.pattern = p
	}

	command := state.InterpolatedCommand()
	exe, args, err := parseCommand(state.path, command)
	if err != nil {
		return err
	}
	debug("executing %s %q", exe, args)
	buf := bytes.NewBuffer(nil)
	cmd := exec.Command(exe, args...)
	cmd.Dir = state.path
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
		err := fmt.Errorf("deadline exceeded by linter %s on %s (try increasing --deadline)", state.name, state.path)
		kerr := cmd.Process.Kill()
		if kerr != nil {
			warning("failed to kill %s: %s", state.name, kerr)
		}
		return err
	}

	if err != nil {
		debug("warning: %s returned %s", command, err)
	}

	processOutput(state, buf.Bytes())
	elapsed := time.Now().Sub(start)
	debug("%s linter took %s", state.name, elapsed)
	return nil
}

func (l *linterState) fixPath(path string) string {
	abspath, err := filepath.Abs(l.path)
	if filepath.IsAbs(path) {
		if err == nil && strings.HasPrefix(path, abspath) {
			normalised := filepath.Join(abspath, filepath.Base(path))
			if _, err := os.Stat(normalised); err == nil {
				path := filepath.Join(l.path, filepath.Base(path))
				return path
			}
		}
	} else {
		return filepath.Join(l.path, path)
	}
	return path
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

		issue := &Issue{Line: 1}
		issue.Linter = LinterFromName(state.name)
		for i, name := range re.SubexpNames() {
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
		if m, ok := linterMessageOverrideFlag[state.name]; ok {
			issue.Message = state.vars.Replace(m)
		}
		if sev, ok := linterSeverityFlag[state.name]; ok {
			issue.Severity = Severity(sev)
		} else {
			issue.Severity = "error"
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
