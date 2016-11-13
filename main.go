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
	"text/template"
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
	// Locations to look for vendored linters.
	vendoredSearchPaths = [][]string{
		{"github.com", "alecthomas", "gometalinter", "vendor"},
		{"gopkg.in", "alecthomas", "gometalinter.v1", "vendor"},
	}
	predefinedPatterns = map[string]string{
		"PATH:LINE:COL:MESSAGE": `^(?P<path>.*?\.go):(?P<line>\d+):(?P<col>\d+):\s*(?P<message>.*)$`,
		"PATH:LINE:MESSAGE":     `^(?P<path>.*?\.go):(?P<line>\d+):\s*(?P<message>.*)$`,
	}
	vetRe       = `^(?:vet:.*?\.go:\s+(?P<path>.*?\.go):(?P<line>\d+):(?P<col>\d+):\s*(?P<message>.*))|(?:(?P<path>.*?\.go):(?P<line>\d+):\s*(?P<message>.*))$`
	lintersFlag = map[string]string{
		"aligncheck":  `aligncheck {path}:^(?:[^:]+: )?(?P<path>.*?\.go):(?P<line>\d+):(?P<col>\d+):\s*(?P<message>.+)$`,
		"deadcode":    `deadcode {path}:^deadcode: (?P<path>.*?\.go):(?P<line>\d+):(?P<col>\d+):\s*(?P<message>.*)$`,
		"dupl":        `dupl -plumbing -threshold {duplthreshold} {path}/*.go:^(?P<path>.*?\.go):(?P<line>\d+)-\d+:\s*(?P<message>.*)$`,
		"errcheck":    `errcheck -abspath {path}:PATH:LINE:COL:MESSAGE`,
		"gas":         `gas -fmt=csv {path}/*.go:^(?P<path>.*?\.go),(?P<line>\d+),(?P<message>[^,]+,[^,]+,[^,]+)`,
		"goconst":     `goconst -min-occurrences {min_occurrences} -min-length {min_const_length} {path}:PATH:LINE:COL:MESSAGE`,
		"gocyclo":     `gocyclo -over {mincyclo} {path}:^(?P<cyclo>\d+)\s+\S+\s(?P<function>\S+)\s+(?P<path>.*?\.go):(?P<line>\d+):(\d+)$`,
		"gofmt":       `gofmt -l -s {path}/*.go:^(?P<path>.*?\.go)$`,
		"goimports":   `goimports -l {path}/*.go:^(?P<path>.*?\.go)$`,
		"golint":      "golint -min_confidence {min_confidence} {path}:PATH:LINE:COL:MESSAGE",
		"gosimple":    "gosimple {path}:PATH:LINE:COL:MESSAGE",
		"gotype":      "gotype -e {tests=-a} {path}:PATH:LINE:COL:MESSAGE",
		"ineffassign": `ineffassign -n {path}:PATH:LINE:COL:MESSAGE`,
		"interfacer":  `interfacer {path}:PATH:LINE:COL:MESSAGE`,
		"lll":         `lll -g -l {maxlinelength} {path}/*.go:PATH:LINE:MESSAGE`,
		"misspell":    "misspell -j 1 {path}/*.go:PATH:LINE:COL:MESSAGE",
		"staticcheck": "staticcheck {path}:PATH:LINE:COL:MESSAGE",
		"structcheck": `structcheck {tests=-t} {path}:^(?:[^:]+: )?(?P<path>.*?\.go):(?P<line>\d+):(?P<col>\d+):\s*(?P<message>.+)$`,
		"test":        `go test {path}:^--- FAIL: .*$\s+(?P<path>.*?\.go):(?P<line>\d+): (?P<message>.*)$`,
		"testify":     `go test {path}:Location:\s+(?P<path>.*?\.go):(?P<line>\d+)$\s+Error:\s+(?P<message>[^\n]+)`,
		"unconvert":   "unconvert {path}:PATH:LINE:COL:MESSAGE",
		"unused":      `unused {path}:PATH:LINE:COL:MESSAGE`,
		"varcheck":    `varcheck {path}:^(?:[^:]+: )?(?P<path>.*?\.go):(?P<line>\d+):(?P<col>\d+):\s*(?P<message>.*)$`,
		"vet":         `go tool vet {path}/*.go:` + vetRe,
		"vetshadow":   `go tool vet --shadow {path}/*.go:` + vetRe,
	}
	disabledLinters           = []string{"testify", "test", "gofmt", "goimports", "lll", "misspell", "unused"}
	enabledLinters            = []string{}
	linterMessageOverrideFlag = map[string]string{
		"errcheck":    "error return value not checked ({message})",
		"gocyclo":     "cyclomatic complexity {cyclo} of function {function}() is high (> {mincyclo})",
		"gofmt":       "file is not gofmted with -s",
		"goimports":   "file is not goimported",
		"structcheck": "unused struct field {message}",
		"varcheck":    "unused global variable {message}",
	}
	linterSeverityFlag = map[string]string{
		"gotype":  "error",
		"test":    "error",
		"testify": "error",
		"vet":     "error",
	}
	installMap = map[string]string{
		"aligncheck":  "github.com/opennota/check/cmd/aligncheck",
		"deadcode":    "github.com/tsenart/deadcode",
		"dupl":        "github.com/mibk/dupl",
		"errcheck":    "github.com/kisielk/errcheck",
		"gas":         "github.com/HewlettPackard/gas",
		"goconst":     "github.com/jgautheron/goconst/cmd/goconst",
		"gocyclo":     "github.com/alecthomas/gocyclo",
		"goimports":   "golang.org/x/tools/cmd/goimports",
		"golint":      "github.com/golang/lint/golint",
		"gosimple":    "honnef.co/go/simple/cmd/gosimple",
		"gotype":      "golang.org/x/tools/cmd/gotype",
		"ineffassign": "github.com/gordonklaus/ineffassign",
		"interfacer":  "github.com/mvdan/interfacer/cmd/interfacer",
		"lll":         "github.com/walle/lll/cmd/lll",
		"misspell":    "github.com/client9/misspell/cmd/misspell",
		"staticcheck": "honnef.co/go/staticcheck/cmd/staticcheck",
		"structcheck": "github.com/opennota/check/cmd/structcheck",
		"unconvert":   "github.com/mdempsky/unconvert",
		"unused":      "honnef.co/go/unused/cmd/unused",
		"varcheck":    "github.com/opennota/check/cmd/varcheck",
	}
	acceptsEllipsis = map[string]bool{
		"aligncheck":  true,
		"errcheck":    true,
		"golint":      true,
		"gosimple":    true,
		"interfacer":  true,
		"staticcheck": true,
		"structcheck": true,
		"test":        true,
		"varcheck":    true,
		"unconvert":   true,
	}
	slowLinters    = []string{"structcheck", "varcheck", "errcheck", "aligncheck", "testify", "test", "interfacer", "unconvert", "deadcode"}
	sortKeys       = []string{"none", "path", "line", "column", "severity", "message", "linter"}
	formatFlag     = "{{.Path}}:{{.Line}}:{{if .Col}}{{.Col}}{{end}}:{{.Severity}}: {{.Message}} ({{.Linter}})"
	formatTemplate = &template.Template{}

	pathsArg            = kingpin.Arg("path", "Directory to lint. Defaults to \".\". <path>/... will recurse.").Strings()
	vendoredLintersFlag = kingpin.Flag("vendored-linters", "Use vendored linters (recommended).").Default("true").Bool()
	fastFlag            = kingpin.Flag("fast", "Only run fast linters.").Bool()
	installFlag         = kingpin.Flag("install", "Attempt to install all known linters.").Short('i').Bool()
	updateFlag          = kingpin.Flag("update", "Pass -u to go tool when installing.").Short('u').Bool()
	forceFlag           = kingpin.Flag("force", "Pass -f to go tool when installing.").Short('f').Bool()
	debugFlag           = kingpin.Flag("debug", "Display messages for failed linters, etc.").Short('d').Bool()
	concurrencyFlag     = kingpin.Flag("concurrency", "Number of concurrent linters to run.").Default("16").Short('j').Int()
	excludeFlag         = kingpin.Flag("exclude", "Exclude messages matching these regular expressions.").Short('e').PlaceHolder("REGEXP").Strings()
	includeFlag         = kingpin.Flag("include", "Include messages matching these regular expressions.").Short('I').PlaceHolder("REGEXP").Strings()
	skipFlag            = kingpin.Flag("skip", "Skip directories with this name when expanding '...'.").Short('s').PlaceHolder("DIR...").Strings()
	vendorFlag          = kingpin.Flag("vendor", "Enable vendoring support (skips 'vendor' directories and sets GO15VENDOREXPERIMENT=1).").Bool()
	cycloFlag           = kingpin.Flag("cyclo-over", "Report functions with cyclomatic complexity over N (using gocyclo).").Default("10").Int()
	lineLengthFlag      = kingpin.Flag("line-length", "Report lines longer than N (using lll).").Default("80").Int()
	minConfidence       = kingpin.Flag("min-confidence", "Minimum confidence interval to pass to golint.").Default(".80").Float()
	minOccurrences      = kingpin.Flag("min-occurrences", "Minimum occurrences to pass to goconst.").Default("3").Int()
	minConstLength      = kingpin.Flag("min-const-length", "Minimumum constant length.").Default("3").Int()
	duplThresholdFlag   = kingpin.Flag("dupl-threshold", "Minimum token sequence as a clone for dupl.").Default("50").Int()
	sortFlag            = kingpin.Flag("sort", fmt.Sprintf("Sort output by any of %s.", strings.Join(sortKeys, ", "))).Default("none").Enums(sortKeys...)
	testFlag            = kingpin.Flag("tests", "Include test files for linters that support this option").Short('t').Bool()
	deadlineFlag        = kingpin.Flag("deadline", "Cancel linters if they have not completed within this duration.").Default("5s").Duration()
	errorsFlag          = kingpin.Flag("errors", "Only show errors.").Bool()
	jsonFlag            = kingpin.Flag("json", "Generate structured JSON rather than standard line-based output.").Bool()
	checkstyleFlag      = kingpin.Flag("checkstyle", "Generate checkstyle XML rather than standard line-based output.").Bool()
	enableGCFlag        = kingpin.Flag("enable-gc", "Enable GC for linters (useful on large repositories).").Bool()
	aggregateFlag       = kingpin.Flag("aggregate", "Aggregate issues reported by several linters.").Bool()
)

func disableAllLinters(*kingpin.ParseContext) error {
	disabledLinters = []string{}
	for linter := range lintersFlag {
		disabledLinters = append(disabledLinters, linter)
	}
	return nil
}

func enableAllLinters(*kingpin.ParseContext) error {
	disabledLinters = []string{}
	return nil
}

func compileOutputTemplate(*kingpin.ParseContext) error {
	tmpl, err := template.New("output").Parse(formatFlag)
	formatTemplate = tmpl
	return err
}

func init() {
	kingpin.FatalIfError(compileOutputTemplate(nil), "")

	kingpin.Flag("disable", fmt.Sprintf("List of linters to disable (%s).", strings.Join(disabledLinters, ","))).PlaceHolder("LINTER").Short('D').StringsVar(&disabledLinters)
	kingpin.Flag("enable", "Enable previously disabled linters.").PlaceHolder("LINTER").Short('E').StringsVar(&enabledLinters)
	kingpin.Flag("linter", "Specify a linter.").PlaceHolder("NAME:COMMAND:PATTERN").StringMapVar(&lintersFlag)
	kingpin.Flag("message-overrides", "Override message from linter. {message} will be expanded to the original message.").PlaceHolder("LINTER:MESSAGE").StringMapVar(&linterMessageOverrideFlag)
	kingpin.Flag("severity", "Map of linter severities.").PlaceHolder("LINTER:SEVERITY").StringMapVar(&linterSeverityFlag)
	kingpin.Flag("disable-all", "Disable all linters.").Action(disableAllLinters).Bool()
	kingpin.Flag("enable-all", "Enable all linters.").Action(enableAllLinters).Bool()
	kingpin.Flag("format", "Output format.").Default(formatFlag).Action(compileOutputTemplate).StringVar(&formatFlag)
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
	buf := new(bytes.Buffer)
	err := formatTemplate.Execute(buf, i)
	kingpin.FatalIfError(err, "Invalid output format")
	return buf.String()
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
	kingpin.CommandLine.Help = fmt.Sprintf(`Aggregate and normalise the output of a whole bunch of Go linters.

Default linters:

%s

Severity override map (default is "warning"):

%s
`, formatLinters(), formatSeverity())
	kingpin.Parse()
	if !*enableGCFlag {
		if err := os.Setenv("GOGC", "off"); err != nil {
			warning("setenv: %v", err)
		}
	}
	if *vendoredLintersFlag && *installFlag && *updateFlag {
		warning(`Linters are now vendored by default, --update ignored. The original
behaviour can be re-enabled with --no-vendored-linters.

To request an update for a vendored linter file an issue at:
https://github.com/alecthomas/gometalinter/issues/new
`)
		*updateFlag = false
	}
	// Force sorting by path if checkstyle mode is selected
	// !jsonFlag check is required to handle:
	// 	gometalinter --json --checkstyle --sort=severity
	if *checkstyleFlag && !*jsonFlag {
		*sortFlag = []string{"path"}
	}

	configureEnvironment()
	// Default to skipping "vendor" directory if GO15VENDOREXPERIMENT=1 is enabled.
	// TODO(alec): This will probably need to be enabled by default at a later time.
	if os.Getenv("GO15VENDOREXPERIMENT") == "1" || *vendorFlag {
		if err := os.Setenv("GO15VENDOREXPERIMENT", "1"); err != nil {
			warning("setenv: %v", err)
		}
		*skipFlag = append(*skipFlag, "vendor")
		*vendorFlag = true
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
	issues, errch := runLinters(linters, paths, *pathsArg, *concurrencyFlag, exclude, include)
	if *jsonFlag {
		status |= outputToJSON(issues)
	} else if *checkstyleFlag {
		status |= outputToCheckstyle(issues)
	} else {
		status |= outputToConsole(issues)
	}
	for err := range errch {
		warning("%s", err)
		status |= 2
	}
	elapsed := time.Since(start)
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

func runLinters(linters map[string]*Linter, paths, ellipsisPaths []string, concurrency int, exclude *regexp.Regexp, include *regexp.Regexp) (chan *Issue, chan error) {
	errch := make(chan error, len(linters)*(len(paths)+len(ellipsisPaths)))
	concurrencych := make(chan bool, *concurrencyFlag)
	incomingIssues := make(chan *Issue, 1000000)
	processedIssues := maybeSortIssues(maybeAggregateIssues(incomingIssues))
	wg := &sync.WaitGroup{}
	for _, linter := range linters {
		// Recreated in each loop because it is mutated by executeLinter().
		vars := Vars{
			"duplthreshold":    fmt.Sprintf("%d", *duplThresholdFlag),
			"mincyclo":         fmt.Sprintf("%d", *cycloFlag),
			"maxlinelength":    fmt.Sprintf("%d", *lineLengthFlag),
			"min_confidence":   fmt.Sprintf("%f", *minConfidence),
			"min_occurrences":  fmt.Sprintf("%d", *minOccurrences),
			"min_const_length": fmt.Sprintf("%d", *minConstLength),
			"tests":            "",
		}
		if *testFlag {
			vars["tests"] = "-t"
		}
		linterPaths := paths
		// Most linters don't exclude vendor paths when recursing, so we don't use ... paths.
		if acceptsEllipsis[linter.Name] && !*vendorFlag && len(ellipsisPaths) > 0 {
			linterPaths = ellipsisPaths
		}
		for _, path := range linterPaths {
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
	if *vendoredLintersFlag {
		cmd = []string{"install"}
	}
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

func maybeAggregateIssues(issues chan *Issue) chan *Issue {
	if !*aggregateFlag {
		return issues
	}
	return aggregateIssues(issues)
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
	vars := l.vars.Copy()
	if l.ShouldChdir() {
		vars["path"] = "."
	} else {
		vars["path"] = l.path
	}
	return vars.Replace(l.Command)
}

func (l *linterState) ShouldChdir() bool {
	return *vendorFlag || !strings.HasSuffix(l.path, "/...") || !strings.Contains(l.Command, "{path}")
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

func (l *linterState) fixPath(path string) string {
	lpath := strings.TrimSuffix(l.path, "...")
	labspath, _ := filepath.Abs(lpath)
	if !filepath.IsAbs(path) {
		path, _ = filepath.Abs(filepath.Join(labspath, path))
	}
	if strings.HasPrefix(path, labspath) {
		return filepath.Join(lpath, strings.TrimPrefix(path, labspath))
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

func findVendoredLinters() string {
	gopaths := strings.Split(os.Getenv("GOPATH"), string(os.PathListSeparator))
	for _, home := range vendoredSearchPaths {
		for _, p := range gopaths {
			joined := append([]string{p, "src"}, home...)
			vendorRoot := filepath.Join(joined...)
			if _, err := os.Stat(vendorRoot); err == nil {
				return vendorRoot
			}
		}
	}
	return ""

}

// Add all "bin" directories from GOPATH to PATH, as well as GOBIN if set.
func configureEnvironment() {
	gopaths := strings.Split(os.Getenv("GOPATH"), string(os.PathListSeparator))
	paths := strings.Split(os.Getenv("PATH"), string(os.PathListSeparator))
	gobin := os.Getenv("GOBIN")

	if *vendoredLintersFlag && *installFlag {
		vendorRoot := findVendoredLinters()
		if vendorRoot == "" {
			kingpin.Fatalf("could not find vendored linters in %s", os.Getenv("GOPATH"))
		}
		debug("found vendored linters at %s, updating environment", vendorRoot)
		if gobin == "" {
			gobin = filepath.Join(gopaths[0], "bin")
		}
		// "go install" panics when one GOPATH element is beneath another, so we just set
		// our vendor root instead.
		gopaths = []string{vendorRoot}
	}

	for _, p := range gopaths {
		paths = append(paths, filepath.Join(p, "bin"))
	}
	if gobin != "" {
		paths = append([]string{gobin}, paths...)
	}

	path := strings.Join(paths, string(os.PathListSeparator))
	gopath := strings.Join(gopaths, string(os.PathListSeparator))

	if err := os.Setenv("PATH", path); err != nil {
		warning("setenv: %v", err)
	}
	debug("PATH=%s", os.Getenv("PATH"))

	if err := os.Setenv("GOPATH", gopath); err != nil {
		warning("setenv: %v", err)
	}
	debug("GOPATH=%s", os.Getenv("GOPATH"))

	if err := os.Setenv("GOBIN", gobin); err != nil {
		warning("setenv: %v", err)
	}
	debug("GOBIN=%s", os.Getenv("GOBIN"))
}
