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
	Name             string   `json:"name"`
	Command          string   `json:"command"`
	CompositeCommand string   `json:"composite_command,omitempty"`
	Pattern          string   `json:"pattern"`
	InstallFrom      string   `json:"install_from"`
	SeverityOverride Severity `json:"severity,omitempty"`
	MessageOverride  string   `json:"message_override,omitempty"`

	regex *regexp.Regexp
}

func (l *Linter) MarshalJSON() ([]byte, error) {
	return json.Marshal(l.Name)
}

func (l *Linter) String() string {
	return l.Name
}

func LinterFromName(name string) *Linter {
	s := lintersFlag[name]
	parts := strings.SplitN(s, ":", 2)
	pattern := parts[1]
	if p, ok := predefinedPatterns[pattern]; ok {
		pattern = p
	}
	re, err := regexp.Compile("(?m:" + pattern + ")")
	kingpin.FatalIfError(err, "invalid regex for %q", name)
	return &Linter{
		Name:             name,
		Command:          s[0:strings.Index(s, ":")],
		Pattern:          pattern,
		InstallFrom:      installMap[name],
		SeverityOverride: Severity(linterSeverityFlag[name]),
		MessageOverride:  linterMessageOverrideFlag[name],
		regex:            re,
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
		"PATH:LINE:COL:MESSAGE": `^(?P<path>[^\s][^\r\n:]+?\.go):(?P<line>\d+):(?P<col>\d+):\s*(?P<message>.*)$`,
		"PATH:LINE:MESSAGE":     `^(?P<path>[^\s][^\r\n:]+?\.go):(?P<line>\d+):\s*(?P<message>.*)$`,
	}
	lintersFlag = map[string]string{
		"aligncheck":  `aligncheck .:^(?:[^:]+: )?(?P<path>[^:]+):(?P<line>\d+):(?P<col>\d+):\s*(?P<message>.+)$`,
		"deadcode":    `deadcode .:^deadcode: (?P<path>[^:]+):(?P<line>\d+):(?P<col>\d+):\s*(?P<message>.*)$`,
		"dupl":        `dupl -plumbing -threshold {duplthreshold} ./*.go:^(?P<path>[^\s][^:]+?\.go):(?P<line>\d+)-\d+:\s*(?P<message>.*)$`,
		"errcheck":    `errcheck -abspath .:^(?P<path>[^:]+):(?P<line>\d+):(?P<col>\d+)[\s\t]+(?P<message>.*)$`,
		"goconst":     `goconst -min-occurrences {min_occurrences} .:PATH:LINE:COL:MESSAGE`,
		"gocyclo":     `gocyclo -over {mincyclo} .:^(?P<cyclo>\d+)\s+\S+\s(?P<function>\S+)\s+(?P<path>[^:]+):(?P<line>\d+):(\d+)$`,
		"gofmt":       `gofmt -l -s ./*.go:^(?P<path>[^\n]+)$`,
		"goimports":   `goimports -l ./*.go:^(?P<path>[^\n]+)$`,
		"golint":      "golint -min_confidence {min_confidence} .:PATH:LINE:COL:MESSAGE",
		"gotype":      "gotype -e {tests=-a} .:PATH:LINE:COL:MESSAGE",
		"ineffassign": `ineffassign -n .:PATH:LINE:COL:MESSAGE`,
		"interfacer":  `interfacer ./:PATH:LINE:COL:MESSAGE`,
		"lll":         `lll -g -l {maxlinelength} ./*.go:PATH:LINE:MESSAGE`,
		"structcheck": `structcheck {tests=-t} .:^(?:[^:]+: )?(?P<path>[^:]+):(?P<line>\d+):(?P<col>\d+):\s*(?P<message>.+)$`,
		"test":        `go test:^--- FAIL: .*$\s+(?P<path>[^:]+):(?P<line>\d+): (?P<message>.*)$`,
		"testify":     `go test:Location:\s+(?P<path>[^:]+):(?P<line>\d+)$\s+Error:\s+(?P<message>[^\n]+)`,
		"varcheck":    `varcheck .:^(?:[^:]+: )?(?P<path>[^:]+):(?P<line>\d+):(?P<col>\d+):[\s\t]+(?P<message>.*)$`,
		"vet":         "go tool vet ./*.go:PATH:LINE:MESSAGE",
		"vetshadow":   "go tool vet --shadow ./*.go:PATH:LINE:MESSAGE",
		"unconvert":   "unconvert .:PATH:LINE:COL:MESSAGE",
		"gosimple":    "gosimple .:PATH:LINE:COL:MESSAGE",
		"staticcheck": "staticcheck .:PATH:LINE:COL:MESSAGE",
		"misspell":    "misspell ./*.go:PATH:LINE:COL:MESSAGE",
	}
	disabledLinters           = []string{"testify", "test", "gofmt", "goimports", "lll", "misspell"}
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
		"gotype":  "error",
		"test":    "error",
		"testify": "error",
		"vet":     "error",
	}
	installMap = map[string]string{
		"golint":      "github.com/golang/lint/golint",
		"gotype":      "golang.org/x/tools/cmd/gotype",
		"goimports":   "golang.org/x/tools/cmd/goimports",
		"errcheck":    "github.com/kisielk/errcheck",
		"varcheck":    "github.com/opennota/check/cmd/varcheck",
		"structcheck": "github.com/opennota/check/cmd/structcheck",
		"aligncheck":  "github.com/opennota/check/cmd/aligncheck",
		"deadcode":    "github.com/tsenart/deadcode",
		"gocyclo":     "github.com/alecthomas/gocyclo",
		"misspell":    "github.com/client9/misspell/cmd/misspell",
		"ineffassign": "github.com/gordonklaus/ineffassign",
		"dupl":        "github.com/mibk/dupl",
		"interfacer":  "github.com/mvdan/interfacer/cmd/interfacer",
		"lll":         "github.com/walle/lll/cmd/lll",
		"unconvert":   "github.com/mdempsky/unconvert",
		"goconst":     "github.com/jgautheron/goconst/cmd/goconst",
		"gosimple":    "honnef.co/go/simple/cmd/gosimple",
		"staticcheck": "honnef.co/go/staticcheck/cmd/staticcheck",
	}
	slowLinters = []string{"structcheck", "varcheck", "errcheck", "aligncheck", "testify", "test", "interfacer", "unconvert"}
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
	lineLengthFlag    = kingpin.Flag("line-length", "Report lines longer than N (using lll).").Default("80").Int()
	minConfidence     = kingpin.Flag("min-confidence", "Minimum confidence interval to pass to golint.").Default(".80").Float()
	minOccurrences    = kingpin.Flag("min-occurrences", "Minimum occurrences to pass to goconst.").Default("3").Int()
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
	// Linters are by their very nature, short lived, so disable GC.
	// Reduced (user) linting time on kingpin from 0.97s to 0.64s.
	_ = os.Setenv("GOGC", "off")
	kingpin.CommandLine.Help = fmt.Sprintf(`Aggregate and normalise the output of a whole bunch of Go linters.

Default linters:

%s

Severity override map (default is "warning"):

%s
`, formatLinters(), formatSeverity())
	kingpin.Parse()
	fixupPath()
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

	if *installFlag {
		installLinters()
		return
	}

	runtime.GOMAXPROCS(*concurrencyFlag)

	start := time.Now()
	paths := expandPaths(*pathsArg, *skipFlag)

	linters := lintersFromFlags()
	status := 0
	issues, errch := runLinters(linters, paths, *concurrencyFlag, exclude, include)
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
		if *errorsFlag && issue.Severity != Error {
			continue
		}
		if status != 0 {
			fmt.Printf(",\n")
		}
		d, err := json.Marshal(issue)
		kingpin.FatalIfError(err, "")
		fmt.Printf("  %s", d)
		status = 1
	}
	fmt.Printf("\n]\n")
	return status
}

func runLinters(linters map[string]*Linter, paths []string, concurrency int, exclude *regexp.Regexp, include *regexp.Regexp) (chan *Issue, chan error) {
	errch := make(chan error, len(linters)*len(paths))
	concurrencych := make(chan bool, *concurrencyFlag)
	incomingIssues := make(chan *Issue, 1000000)
	processedIssues := maybeSortIssues(incomingIssues)
	wg := &sync.WaitGroup{}
	for _, linter := range linters {
		// Recreated in each loop because it is mutated by executeLinter().
		vars := Vars{
			"duplthreshold":   fmt.Sprintf("%d", *duplThresholdFlag),
			"mincyclo":        fmt.Sprintf("%d", *cycloFlag),
			"maxlinelength":   fmt.Sprintf("%d", *lineLengthFlag),
			"min_confidence":  fmt.Sprintf("%f", *minConfidence),
			"min_occurrences": fmt.Sprintf("%d", *minOccurrences),
			"tests":           "",
		}
		if *testFlag {
			vars["tests"] = "-t"
		}
		for _, path := range paths {
			wg.Add(1)
			deadline := time.After(*deadlineFlag)
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

func makeInstallCommand(linters ...string) []string {
	cmd := []string{"get"}
	if *debugFlag {
		cmd = append(cmd, "-v")
	}
	if *updateFlag {
		cmd = append(cmd, "-u")
	}
	if *forceFlag {
		cmd = append(cmd, "-f")
	}
	cmd = append(cmd, linters...)
	return cmd
}

func installLintersWithOneCommand(targets []string) error {
	cmd := makeInstallCommand(targets...)
	debug("go %s", strings.Join(cmd, " "))
	c := exec.Command("go", cmd...)
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Run()
}

func installLintersIndividually(targets []string) {
	failed := []string{}
	for _, target := range targets {
		cmd := makeInstallCommand(target)
		debug("go %s", strings.Join(cmd, " "))
		c := exec.Command("go", cmd...)
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr
		if err := c.Run(); err != nil {
			warning("failed to install %s: %s", target, err)
			failed = append(failed, target)
		}
	}
	if len(failed) > 0 {
		kingpin.Fatalf("failed to install the following linters: %s", strings.Join(failed, ", "))
	}
}

func installLinters() {
	names := make([]string, 0, len(installMap))
	targets := make([]string, 0, len(installMap))
	for name, target := range installMap {
		names = append(names, name)
		targets = append(targets, target)
	}
	namesStr := strings.Join(names, "\n  ")
	fmt.Printf("Installing:\n  %s\n", namesStr)
	err := installLintersWithOneCommand(targets)
	if err == nil {
		return
	}
	warning("failed to install one or more linters: %s (installing individually)", err)
	installLintersIndividually(targets)
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
	*Linter
	path     string
	issues   chan *Issue
	vars     Vars
	exclude  *regexp.Regexp
	include  *regexp.Regexp
	deadline <-chan time.Time
}

func (l *linterState) InterpolatedCommand() string {
	l.vars["path"] = l.path
	return l.vars.Replace(l.Command)
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
		err := fmt.Errorf("deadline exceeded by linter %s on %s (try increasing --deadline)", state.Name, state.path)
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
	elapsed := time.Now().Sub(start)
	debug("%s linter took %s", state.Name, elapsed)
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

func lintersFromFlags() map[string]*Linter {
	out := map[string]*Linter{}
	for name := range lintersFlag {
		out[name] = LinterFromName(name)
	}
	enabled := make([]string, len(enabledLinters))
	copy(enabled, enabledLinters)
	disabled := make([]string, len(disabledLinters))
	copy(disabled, disabledLinters)

	// Disable slow linters.
	if *fastFlag {
		disabled = append(disabled, slowLinters...)
	}

	disable := map[string]bool{}
	for _, linter := range disabled {
		disable[linter] = true
	}
	for _, linter := range enabled {
		delete(disable, linter)
	}

	for linter := range disable {
		delete(out, linter)
	}
	return out
}

func processOutput(state *linterState, out []byte) {
	re := state.regex
	all := re.FindAllSubmatchIndex(out, -1)
	debug("%s hits %d: %s", state.Name, len(all), state.Pattern)
	for _, indices := range all {
		group := [][]byte{}
		for i := 0; i < len(indices); i += 2 {
			fragment := out[indices[i]:indices[i+1]]
			group = append(group, fragment)
		}

		issue := &Issue{Line: 1}
		issue.Linter = LinterFromName(state.Name)
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
		if m, ok := linterMessageOverrideFlag[state.Name]; ok {
			issue.Message = state.vars.Replace(m)
		}
		if sev, ok := linterSeverityFlag[state.Name]; ok {
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

// Add all "bin" directories from GOPATH to PATH.
func fixupPath() {
	paths := strings.Split(os.Getenv("PATH"), string(os.PathListSeparator))
	gopaths := strings.Split(os.Getenv("GOPATH"), string(os.PathListSeparator))
	for i, p := range gopaths {
		gopaths[i] = filepath.Join(p, "bin")
	}
	paths = append(gopaths, paths...)
	path := strings.Join(paths, string(os.PathListSeparator))
	os.Setenv("PATH", path)
	debug("PATH=%s", os.Getenv("PATH"))
}
