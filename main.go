package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"text/template"
	"time"

	"gopkg.in/alecthomas/kingpin.v3-unstable"
)

var (
	// Locations to look for vendored linters.
	vendoredSearchPaths = [][]string{
		{"github.com", "alecthomas", "gometalinter", "_linters"},
		{"gopkg.in", "alecthomas", "gometalinter.v1", "_linters"},
	}
)

func init() {
	kingpin.Flag("config", "Load JSON configuration from file.").Action(loadConfig).String()
	kingpin.Flag("disable", "Disable previously enabled linters.").PlaceHolder("LINTER").Short('D').Action(disableAction).Strings()
	kingpin.Flag("enable", "Enable previously disabled linters.").PlaceHolder("LINTER").Short('E').Action(enableAction).Strings()
	kingpin.Flag("linter", "Define a linter.").PlaceHolder("NAME:COMMAND:PATTERN").StringMapVar(&config.Linters)
	kingpin.Flag("message-overrides", "Override message from linter. {message} will be expanded to the original message.").PlaceHolder("LINTER:MESSAGE").StringMapVar(&config.MessageOverride)
	kingpin.Flag("severity", "Map of linter severities.").PlaceHolder("LINTER:SEVERITY").StringMapVar(&config.Severity)
	kingpin.Flag("disable-all", "Disable all linters.").Action(disableAllAction).Bool()
	kingpin.Flag("enable-all", "Enable all linters.").Action(enableAllAction).Bool()
	kingpin.Flag("format", "Output format.").PlaceHolder(config.Format).StringVar(&config.Format)
	kingpin.Flag("vendored-linters", "Use vendored linters (recommended).").BoolVar(&config.VendoredLinters)
	kingpin.Flag("fast", "Only run fast linters.").BoolVar(&config.Fast)
	kingpin.Flag("install", "Attempt to install all known linters.").Short('i').BoolVar(&config.Install)
	kingpin.Flag("update", "Pass -u to go tool when installing.").Short('u').BoolVar(&config.Update)
	kingpin.Flag("force", "Pass -f to go tool when installing.").Short('f').BoolVar(&config.Force)
	kingpin.Flag("download-only", "Pass -d to go tool when installing.").BoolVar(&config.DownloadOnly)
	kingpin.Flag("debug", "Display messages for failed linters, etc.").Short('d').BoolVar(&config.Debug)
	kingpin.Flag("concurrency", "Number of concurrent linters to run.").PlaceHolder(fmt.Sprintf("%d", runtime.NumCPU())).Short('j').IntVar(&config.Concurrency)
	kingpin.Flag("exclude", "Exclude messages matching these regular expressions.").Short('e').PlaceHolder("REGEXP").StringsVar(&config.Exclude)
	kingpin.Flag("include", "Include messages matching these regular expressions.").Short('I').PlaceHolder("REGEXP").StringsVar(&config.Include)
	kingpin.Flag("skip", "Skip directories with this name when expanding '...'.").Short('s').PlaceHolder("DIR...").StringsVar(&config.Skip)
	kingpin.Flag("vendor", "Enable vendoring support (skips 'vendor' directories and sets GO15VENDOREXPERIMENT=1).").BoolVar(&config.Vendor)
	kingpin.Flag("cyclo-over", "Report functions with cyclomatic complexity over N (using gocyclo).").PlaceHolder("10").IntVar(&config.Cyclo)
	kingpin.Flag("line-length", "Report lines longer than N (using lll).").PlaceHolder("80").IntVar(&config.LineLength)
	kingpin.Flag("min-confidence", "Minimum confidence interval to pass to golint.").PlaceHolder(".80").FloatVar(&config.MinConfidence)
	kingpin.Flag("min-occurrences", "Minimum occurrences to pass to goconst.").PlaceHolder("3").IntVar(&config.MinOccurrences)
	kingpin.Flag("min-const-length", "Minimumum constant length.").PlaceHolder("3").IntVar(&config.MinConstLength)
	kingpin.Flag("dupl-threshold", "Minimum token sequence as a clone for dupl.").PlaceHolder("50").IntVar(&config.DuplThreshold)
	kingpin.Flag("sort", fmt.Sprintf("Sort output by any of %s.", strings.Join(sortKeys, ", "))).PlaceHolder("none").EnumsVar(&config.Sort, sortKeys...)
	kingpin.Flag("tests", "Include test files for linters that support this option").Short('t').BoolVar(&config.Test)
	kingpin.Flag("deadline", "Cancel linters if they have not completed within this duration.").PlaceHolder("30s").DurationVar(&config.Deadline)
	kingpin.Flag("errors", "Only show errors.").BoolVar(&config.Errors)
	kingpin.Flag("json", "Generate structured JSON rather than standard line-based output.").BoolVar(&config.JSON)
	kingpin.Flag("checkstyle", "Generate checkstyle XML rather than standard line-based output.").BoolVar(&config.Checkstyle)
	kingpin.Flag("enable-gc", "Enable GC for linters (useful on large repositories).").BoolVar(&config.EnableGC)
	kingpin.Flag("aggregate", "Aggregate issues reported by several linters.").BoolVar(&config.Aggregate)
	kingpin.CommandLine.GetFlag("help").Short('h')
}

func loadConfig(app *kingpin.Application, element *kingpin.ParseElement, ctx *kingpin.ParseContext) error {
	r, err := os.Open(*element.Value)
	if err != nil {
		return err
	}
	defer r.Close() // nolint: errcheck
	err = json.NewDecoder(r).Decode(config)
	if err != nil {
		return err
	}
	if config.DeadlineJSONCrutch != "" {
		config.Deadline, err = time.ParseDuration(config.DeadlineJSONCrutch)
	}
	for _, disable := range config.Disable {
		for i, enable := range config.Enable {
			if enable == disable {
				config.Enable = append(config.Enable[:i], config.Enable[i+1:]...)
				break
			}
		}
	}
	return err
}

func disableAction(app *kingpin.Application, element *kingpin.ParseElement, ctx *kingpin.ParseContext) error {
	out := []string{}
	for _, linter := range config.Enable {
		if linter != *element.Value {
			out = append(out, linter)
		}
	}
	config.Enable = out
	return nil
}

func enableAction(app *kingpin.Application, element *kingpin.ParseElement, ctx *kingpin.ParseContext) error {
	config.Enable = append(config.Enable, *element.Value)
	return nil
}

func disableAllAction(app *kingpin.Application, element *kingpin.ParseElement, ctx *kingpin.ParseContext) error {
	config.Enable = []string{}
	return nil
}

func enableAllAction(app *kingpin.Application, element *kingpin.ParseElement, ctx *kingpin.ParseContext) error {
	for linter := range linterDefinitions {
		config.Enable = append(config.Enable, linter)
	}
	return nil
}

func debug(format string, args ...interface{}) {
	if config.Debug {
		fmt.Fprintf(os.Stderr, "DEBUG: "+format+"\n", args...)
	}
}

func warning(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "WARNING: "+format+"\n", args...)
}

func formatLinters() string {
	w := bytes.NewBuffer(nil)
	for name := range linterDefinitions {
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
	for name, severity := range config.Severity {
		fmt.Fprintf(w, "  %s -> %s\n", name, severity)
	}
	return w.String()
}

func main() {
	var pathsArg = kingpin.Arg("path", "Directories to lint. Defaults to \".\". <path>/... will recurse.").Strings()
	kingpin.CommandLine.Help = fmt.Sprintf(`Aggregate and normalise the output of a whole bunch of Go linters.

PlaceHolder linters:

%s

Severity override map (default is "warning"):

%s
`, formatLinters(), formatSeverity())
	kingpin.Parse()

	configureEnvironment()

	if config.Install {
		installLinters()
		return
	}

	include, exclude := processConfig(config)

	start := time.Now()
	paths := resolvePaths(*pathsArg, config.Skip)

	linters := lintersFromFlags()
	status := 0
	issues, errch := runLinters(linters, paths, config.Concurrency, exclude, include)
	if config.JSON {
		status |= outputToJSON(issues)
	} else if config.Checkstyle {
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

// nolint: gocyclo
func processConfig(config *Config) (include *regexp.Regexp, exclude *regexp.Regexp) {
	// Move configured linters into linterDefinitions.
	for name, definition := range config.Linters {
		linterDefinitions[name] = definition
	}

	tmpl, err := template.New("output").Parse(config.Format)
	kingpin.FatalIfError(err, "invalid format %q", config.Format)
	formatTemplate = tmpl

	if !config.EnableGC {
		// Linters are by their very nature, short lived, so disable GC.
		// Reduced (user) linting time on kingpin from 0.97s to 0.64s.
		_ = os.Setenv("GOGC", "off")
	}
	if config.VendoredLinters && config.Install && config.Update {
		warning(`Linters are now vendored by default, --update ignored. The original
behaviour can be re-enabled with --no-vendored-linters.

To request an update for a vendored linter file an issue at:
https://github.com/alecthomas/gometalinter/issues/new
`)
		config.Update = false
	}
	// Force sorting by path if checkstyle mode is selected
	// !jsonFlag check is required to handle:
	// 	gometalinter --json --checkstyle --sort=severity
	if config.Checkstyle && !config.JSON {
		config.Sort = []string{"path"}
	}

	// PlaceHolder to skipping "vendor" directory if GO15VENDOREXPERIMENT=1 is enabled.
	// TODO(alec): This will probably need to be enabled by default at a later time.
	if os.Getenv("GO15VENDOREXPERIMENT") == "1" || config.Vendor {
		if err := os.Setenv("GO15VENDOREXPERIMENT", "1"); err != nil {
			warning("setenv GO15VENDOREXPERIMENT: %s", err)
		}
		config.Skip = append(config.Skip, "vendor")
		config.Vendor = true
	}
	if len(config.Exclude) > 0 {
		exclude = regexp.MustCompile(strings.Join(config.Exclude, "|"))
	}

	if len(config.Include) > 0 {
		include = regexp.MustCompile(strings.Join(config.Include, "|"))
	}

	runtime.GOMAXPROCS(config.Concurrency)
	return include, exclude
}

func outputToConsole(issues chan *Issue) int {
	status := 0
	for issue := range issues {
		if config.Errors && issue.Severity != Error {
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
		if config.Errors && issue.Severity != Error {
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

func resolvePaths(paths, skip []string) []string {
	if len(paths) == 0 {
		return []string{"."}
	}

	skipPath := newPathFilter(skip)
	dirs := map[string]bool{}
	for _, path := range paths {
		if strings.HasSuffix(path, "/...") {
			root := filepath.Dir(path)
			_ = filepath.Walk(root, func(p string, i os.FileInfo, err error) error {
				if err != nil {
					warning("invalid path %q: %s", p, err)
					return err
				}

				skip := skipPath(p)
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
		out = append(out, relativePackagePath(d))
	}
	sort.Strings(out)
	for _, d := range out {
		debug("linting path %s", d)
	}
	return out
}

func newPathFilter(skip []string) func(string) bool {
	filter := map[string]bool{}
	for _, name := range skip {
		filter[name] = true
	}

	return func(path string) bool {
		base := filepath.Base(path)
		if filter[base] || filter[path] {
			return true
		}
		return base != "." && base != ".." && strings.ContainsAny(base[0:1], "_.")
	}
}

func relativePackagePath(dir string) string {
	if filepath.IsAbs(dir) || strings.HasPrefix(dir, ".") {
		return dir
	}
	// package names must start with a ./
	return "./" + dir
}

func lintersFromFlags() map[string]*Linter {
	out := map[string]*Linter{}
	config.Enable = replaceWithMegacheck(config.Enable)
	for _, linter := range config.Enable {
		out[linter] = LinterFromName(linter)
	}
	for _, linter := range config.Disable {
		delete(out, linter)
	}
	if config.Fast {
		for _, linter := range slowLinters {
			delete(out, linter)
		}
	}
	return out
}

// replaceWithMegacheck checks enabled linters if they duplicate megacheck and
// returns a either a revised list removing those and adding megacheck or an
// unchanged slice. Emits a warning if linters were removed and swapped with
// megacheck.
func replaceWithMegacheck(enabled []string) []string {
	var (
		staticcheck,
		gosimple,
		unused bool
		revised []string
	)
	for _, linter := range enabled {
		switch linter {
		case "staticcheck":
			staticcheck = true
		case "gosimple":
			gosimple = true
		case "unused":
			unused = true
		case "megacheck":
			// Don't add to revised slice, we'll add it later
		default:
			revised = append(revised, linter)
		}
	}
	if staticcheck && gosimple && unused {
		warning("staticcheck, gosimple, and unused are all set, using megacheck instead")
		return append(revised, "megacheck")
	}
	return enabled
}

func findVendoredLinters() string {
	gopaths := strings.Split(getGoPath(), string(os.PathListSeparator))
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

// Go 1.8 compatible GOPATH.
func getGoPath() string {
	path := os.Getenv("GOPATH")
	if path == "" {
		user, err := user.Current()
		kingpin.FatalIfError(err, "")
		path = filepath.Join(user.HomeDir, "go")
	}
	return path
}

// addPath appends p to paths and returns it if:
// 1. p is not a blank string
// 2. p doesn't already exist in paths
// Otherwise paths is returned unchanged.
func addPath(p string, paths []string) []string {
	if p == "" {
		return paths
	}
	for _, path := range paths {
		if p == path {
			return paths
		}
	}
	return append(paths, p)
}

// Ensure all "bin" directories from GOPATH exists in PATH, as well as GOBIN if set.
func configureEnvironment() {
	gopaths := strings.Split(getGoPath(), string(os.PathListSeparator))
	paths := strings.Split(os.Getenv("PATH"), string(os.PathListSeparator))
	gobin := os.Getenv("GOBIN")

	if config.VendoredLinters && config.Install {
		vendorRoot := findVendoredLinters()
		if vendorRoot == "" {
			kingpin.Fatalf("could not find vendored linters in GOPATH=%q", getGoPath())
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
		paths = addPath(filepath.Join(p, "bin"), paths)
	}
	paths = addPath(gobin, paths)

	path := strings.Join(paths, string(os.PathListSeparator))
	gopath := strings.Join(gopaths, string(os.PathListSeparator))

	if err := os.Setenv("PATH", path); err != nil {
		warning("setenv PATH: %s", err)
	}
	debug("PATH=%s", os.Getenv("PATH"))

	if err := os.Setenv("GOPATH", gopath); err != nil {
		warning("setenv GOPATH: %s", err)
	}
	debug("GOPATH=%s", os.Getenv("GOPATH"))

	if err := os.Setenv("GOBIN", gobin); err != nil {
		warning("setenv GOBIN: %s", err)
	}
	debug("GOBIN=%s", os.Getenv("GOBIN"))
}
