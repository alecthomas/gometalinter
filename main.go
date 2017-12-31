package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"text/template"
	"time"

	"github.com/kisielk/gotool"
	kingpin "gopkg.in/alecthomas/kingpin.v3-unstable"

	"github.com/alecthomas/gometalinter/output"
	"github.com/alecthomas/gometalinter/pipeline"
	. "github.com/alecthomas/gometalinter/util" // nolint
)

var (
	// Locations to look for vendored linters.
	vendoredSearchPaths = [][]string{
		{"github.com", "alecthomas", "gometalinter", "_linters"},
		{"gopkg.in", "alecthomas", "gometalinter.v2", "_linters"},
	}
	defaultConfigPath = ".gometalinter.json"
	Version           = "master"
)

func setupFlags(app *kingpin.Application) {
	app.Flag("help-man", "Show help as a man page.").Action(showManPage).Bool()
	app.Flag("config", "Load JSON configuration from file.").Envar("GOMETALINTER_CONFIG").Action(loadConfig).String()
	app.Flag("no-config", "Disable automatic loading of config file.").Bool()
	app.Flag("disable", "Disable previously enabled linters.").PlaceHolder("LINTER").Short('D').Action(disableAction).Strings()
	app.Flag("enable", "Enable previously disabled linters.").PlaceHolder("LINTER").Short('E').Action(enableAction).Strings()
	app.Flag("linter", "Define a linter.").PlaceHolder("NAME:COMMAND:PATTERN").Action(cliLinterOverrides).StringMap()
	app.Flag("message-overrides", "Override message from linter. {message} will be expanded to the original message.").PlaceHolder("LINTER:MESSAGE").StringMapVar(&config.MessageOverride)
	app.Flag("severity", "Map of linter severities.").PlaceHolder("LINTER:SEVERITY").StringMapVar(&config.Severity)
	app.Flag("disable-all", "Disable all linters.").Action(disableAllAction).Bool()
	app.Flag("enable-all", "Enable all linters.").Action(enableAllAction).Bool()
	app.Flag("format", "Output format.").PlaceHolder(config.Format).StringVar(&config.Format)
	app.Flag("fast", "Only run fast linters.").BoolVar(&config.Fast)
	app.Flag("debug", "Display messages for failed linters, etc.").Short('d').BoolVar(&config.Debug)
	app.Flag("concurrency", "Number of concurrent linters to run.").PlaceHolder(fmt.Sprintf("%d", runtime.NumCPU())).Short('j').IntVar(&config.Concurrency)
	app.Flag("exclude", "Exclude messages matching these regular expressions.").Short('e').PlaceHolder("REGEXP").StringsVar(&config.Exclude)
	app.Flag("include", "Include messages matching these regular expressions.").Short('I').PlaceHolder("REGEXP").StringsVar(&config.Include)
	app.Flag("skip", "Skip directories with this name.").Short('s').PlaceHolder("DIR...").StringsVar(&config.Skip)
	app.Flag("cyclo-over", "Report functions with cyclomatic complexity over N (using gocyclo).").PlaceHolder("10").IntVar(&config.Cyclo)
	app.Flag("line-length", "Report lines longer than N (using lll).").PlaceHolder("80").IntVar(&config.LineLength)
	app.Flag("misspell-locale", "Specify locale to use (using misspell).").PlaceHolder("").StringVar(&config.MisspellLocale)
	app.Flag("min-confidence", "Minimum confidence interval to pass to golint.").PlaceHolder(".80").FloatVar(&config.MinConfidence)
	app.Flag("min-occurrences", "Minimum occurrences to pass to goconst.").PlaceHolder("3").IntVar(&config.MinOccurrences)
	app.Flag("min-const-length", "Minimum constant length.").PlaceHolder("3").IntVar(&config.MinConstLength)
	app.Flag("dupl-threshold", "Minimum token sequence as a clone for dupl.").PlaceHolder("50").IntVar(&config.DuplThreshold)
	app.Flag("sort", fmt.Sprintf("Sort output by any of %s.", strings.Join(sortKeys, ", "))).PlaceHolder("none").EnumsVar(&config.Sort, sortKeys...)
	app.Flag("tests", "Include test files for linters that support this option.").Short('t').BoolVar(&config.Test)
	app.Flag("deadline", "Cancel linters if they have not completed within this duration.").PlaceHolder("30s").DurationVar((*time.Duration)(&config.Deadline))
	app.Flag("errors", "Only show errors.").BoolVar(&config.Errors)
	app.Flag("json", "Generate structured JSON rather than standard line-based output.").BoolVar(&config.JSON)
	app.Flag("checkstyle", "Generate checkstyle XML rather than standard line-based output.").BoolVar(&config.Checkstyle)
	app.Flag("enable-gc", "Enable GC for linters (useful on large repositories).").BoolVar(&config.EnableGC)
	app.Flag("aggregate", "Aggregate issues reported by several linters.").BoolVar(&config.Aggregate)
	app.Flag("warn-unmatched-nolint", "Warn if a nolint directive is not matched with an issue.").BoolVar(&config.WarnUnmatchedDirective)
	app.GetFlag("help").Short('h')
}

func showManPage(app *kingpin.Application, element *kingpin.ParseElement, context *kingpin.ParseContext) error {
	app.UsageTemplate(kingpin.ManPageTemplate).Usage(nil)
	os.Exit(0)
	return nil
}

func cliLinterOverrides(app *kingpin.Application, element *kingpin.ParseElement, ctx *kingpin.ParseContext) error {
	// expected input structure - <name>:<command-spec>
	parts := strings.SplitN(*element.Value, ":", 2)
	if len(parts) < 2 {
		return fmt.Errorf("incorrectly formatted input: %s", *element.Value)
	}
	name := parts[0]
	spec := parts[1]
	conf, err := parseLinterConfigSpec(name, spec)
	if err != nil {
		return fmt.Errorf("incorrectly formatted input: %s", *element.Value)
	}
	config.Linters[name] = StringOrLinterConfig(conf)
	return nil
}

func loadDefaultConfig(app *kingpin.Application, element *kingpin.ParseElement, ctx *kingpin.ParseContext) error {
	if element != nil {
		return nil
	}

	for _, elem := range ctx.Elements {
		if f := elem.OneOf.Flag; f == app.GetFlag("config") || f == app.GetFlag("no-config") {
			return nil
		}
	}

	configFile, found, err := findDefaultConfigFile()
	if err != nil || !found {
		return err
	}

	return loadConfigFile(configFile)
}

func loadConfig(app *kingpin.Application, element *kingpin.ParseElement, ctx *kingpin.ParseContext) error {
	return loadConfigFile(*element.Value)
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
	for linter := range defaultLinters {
		config.Enable = append(config.Enable, linter)
	}
	config.EnableAll = true
	return nil
}

func formatLinters() string {
	w := bytes.NewBuffer(nil)
	for _, linter := range getDefaultLinters() {
		install := "(" + linter.InstallFrom + ")"
		if install == "()" {
			install = ""
		}
		fmt.Fprintf(w, "  %s: %s\n\tcommand: %s\n\tregex: %s\n\tfast: %t\n\tdefault enabled: %t\n\n",
			linter.Name, install, linter.Command, linter.Pattern, linter.IsFast, linter.defaultEnabled)
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
	kingpin.Version(Version)
	pathsArg := kingpin.Arg("path", "Directories to lint. Defaults to \".\". <path>/... will recurse.").Strings()
	app := kingpin.CommandLine
	app.Action(loadDefaultConfig)
	setupFlags(app)
	app.Help = fmt.Sprintf(`Aggregate and normalise the output of a whole bunch of Go linters.

PlaceHolder linters:

%s

Severity override map (default is "warning"):

%s
`, formatLinters(), formatSeverity())
	kingpin.Parse()

	Debugging(config.Debug)

	configureEnvironment()
	include, exclude := processConfig(config)

	start := time.Now()
	paths := resolvePaths(*pathsArg, config.Skip)

	linters := lintersFromConfig(config)
	err := validateLinters(linters, config)
	kingpin.FatalIfError(err, "")

	issues, errch := runLinters(linters, paths, config.Concurrency, exclude, include)
	issueStatus, issues := pipeline.Status(issues)
	status := 0
	if config.JSON {
		err = output.JSON(os.Stdout, issues)
	} else if config.Checkstyle {
		err = output.Checkstyle(os.Stdout, issues)
	} else {
		err = output.Text(os.Stdout, config.formatTemplate, issues)
	}
	if err != nil {
		Warning("%s", err)
		status |= 4
	}
	for err := range errch {
		Warning("%s", err)
		status |= 4
	}
	elapsed := time.Since(start)
	status |= <-issueStatus
	Debug("total elapsed time %s", elapsed)
	os.Exit(status)
}

// nolint: gocyclo
func processConfig(config *Config) (include *regexp.Regexp, exclude *regexp.Regexp) {
	tmpl, err := template.New("output").Parse(config.Format)
	kingpin.FatalIfError(err, "invalid format %q", config.Format)
	config.formatTemplate = tmpl

	// Linters are by their very nature, short lived, so disable GC.
	// Reduced (user) linting time on kingpin from 0.97s to 0.64s.
	if !config.EnableGC {
		_ = os.Setenv("GOGC", "off")
	}
	// Force sorting by path if checkstyle mode is selected
	// !jsonFlag check is required to handle:
	// 	gometalinter --json --checkstyle --sort=severity
	if config.Checkstyle && !config.JSON {
		config.Sort = []string{"path"}
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

func resolvePaths(paths, skip []string) []string {
	if len(paths) == 0 {
		return []string{"."}
	}

	skipPath := newPathFilter(skip)
	dirs := newStringSet()
	for _, dir := range gotool.ImportPaths(paths) {
		if !skipPath(dir) {
			dirs.add(relativePackagePath(dir))
		}
	}
	out := dirs.asSlice()
	sort.Strings(out)
	for _, d := range out {
		Debug("linting path %s", d)
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

func lintersFromConfig(config *Config) map[string]*Linter {
	out := map[string]*Linter{}
	for _, name := range config.Enable {
		linter := getLinterByName(name, LinterConfig(config.Linters[name]))
		if config.Fast && !linter.IsFast {
			continue
		}
		out[name] = linter
	}
	for _, linter := range config.Disable {
		delete(out, linter)
	}
	return out
}

func findVendoredLinters() string {
	gopaths := getGoPathList()
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

// addPath appends path to paths if path does not already exist in paths. Returns
// the new paths.
func addPath(paths []string, path string) []string {
	for _, existingpath := range paths {
		if path == existingpath {
			return paths
		}
	}
	return append(paths, path)
}

// configureEnvironment adds all `bin/` directories from $GOPATH to $PATH
func configureEnvironment() {
	paths := addGoBinsToPath(getGoPathList())
	setEnv("PATH", strings.Join(paths, string(os.PathListSeparator)))
	debugPrintEnv()
}

func addGoBinsToPath(gopaths []string) []string {
	paths := strings.Split(os.Getenv("PATH"), string(os.PathListSeparator))
	for _, p := range gopaths {
		paths = addPath(paths, filepath.Join(p, "bin"))
	}
	gobin := os.Getenv("GOBIN")
	if gobin != "" {
		paths = addPath(paths, gobin)
	}
	return paths
}

func setEnv(key string, value string) {
	if err := os.Setenv(key, value); err != nil {
		Warning("setenv %s: %s", key, err)
	}
}

func debugPrintEnv() {
	Debug("PATH=%s", os.Getenv("PATH"))
	Debug("GOPATH=%s", os.Getenv("GOPATH"))
	Debug("GOBIN=%s", os.Getenv("GOBIN"))
}
