package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/alecthomas/kingpin"
)

type Severity string

// Linter message severity levels.
const (
	Warning Severity = "warning"
	Error   Severity = "error"
)

var (
	predefinedPatterns = map[string]string{
		"PATH:LINE:COL:MESSAGE": `(?P<path>[^:]+):(?P<line>\d+):(?P<col>\d+):\s*(?P<message>.*)`,
		"PATH:LINE:MESSAGE":     `(?P<path>[^:]+):(?P<line>\d+):\s*(?P<message>.*)`,
	}
	lintersFlag = map[string]string{
		// main.go:8:10: should omit type map[string]string from declaration of var linters; it will be inferred from the right-hand side
		"golint": "golint {paths}:PATH:LINE:COL:MESSAGE",
		// test/stutter.go:19: missing argument for Printf("%d"): format reads arg 1, have only 0 args
		"vet":        "go tool vet {paths}:PATH:LINE:MESSAGE",
		"gotype":     "gotype {paths}:PATH:LINE:COL:MESSAGE",
		"errcheck":   `errcheck {paths}:(?P<path>[^:]+):(?P<line>\d+):(?P<col>\d+)\t(?P<message>.*)`,
		"varcheck":   "varcheck {paths}:PATH:LINE:MESSAGE",
		"defercheck": "defercheck {paths}:PATH:LINE:MESSAGE",
	}
	linterMessageOverrideFlag = map[string]string{
		"errcheck": "error return value not checked ({message})",
		"varcheck": "unused global variable {message}",
	}
	linterSeverityFlag = map[string]string{
		"errcheck": "warning",
		"golint":   "warning",
	}
	pathsArg           = kingpin.Arg("paths", "Directories to lint.").Required().Strings()
	disableLintersFlag = kingpin.Flag("disable-linters", "List of linters to disable.").PlaceHolder("LINTER").Strings()
	debugFlag          = kingpin.Flag("debug", "Display messages for failed linters, etc.").Bool()
	concurrencyFlag    = kingpin.Flag("concurrency", "Number of concurrent linters to run.").Default("16").Int()
)

func init() {
	kingpin.Flag("linter", "Specify a linter.").PlaceHolder("NAME:COMMAND:PATTERN").StringMapVar(&lintersFlag)
	kingpin.Flag("linter-message-overrides", "Override message from linter.").PlaceHolder("LINTER:MESSAGE").StringMapVar(&linterMessageOverrideFlag)
	kingpin.Flag("linter-severity", "Map of linter severities.").PlaceHolder("LINTER:SEVERITY").StringMapVar(&linterSeverityFlag)
}

type Issue struct {
	severity Severity
	path     string
	line     int
	col      int
	message  string
}

func (m *Issue) String() string {
	col := ""
	if m.col != 0 {
		col = fmt.Sprintf("%d", m.col)
	}
	return fmt.Sprintf("%s:%d:%s:%s: %s", m.path, m.line, col, m.severity, m.message)
}

type Issues []*Issue

func (m Issues) Len() int      { return len(m) }
func (m Issues) Swap(i, j int) { m[i], m[j] = m[j], m[i] }
func (m Issues) Less(i, j int) bool {
	return m[i].path < m[j].path || m[i].line < m[j].line || m[i].col < m[j].col
}

func debug(format string, args ...interface{}) {
	if *debugFlag {
		fmt.Fprintf(os.Stderr, "DEBUG: "+format+"\n", args...)
	}
}

func formatLinters() string {
	out := &bytes.Buffer{}
	for command, description := range lintersFlag {
		parts := strings.SplitN(description, ":", 2)
		fmt.Fprintf(out, "    %s -> %s -> %s\n", command, parts[0], parts[1])
	}
	return out.String()
}

func main() {
	kingpin.CommandLine.Help = fmt.Sprintf(`Aggregate and normalise the output of a whole bunch of Go linters.

Default linters:

%s
`, formatLinters())
	kingpin.Parse()
	runtime.GOMAXPROCS(*concurrencyFlag)

	disable := map[string]bool{}
	for _, linter := range *disableLintersFlag {
		disable[linter] = true
	}

	start := time.Now()

	paths := strings.Join(*pathsArg, " ")
	concurrency := make(chan bool, *concurrencyFlag)
	issues := make(chan *Issue, 1000)
	wg := &sync.WaitGroup{}
	for name, description := range lintersFlag {
		if _, ok := disable[name]; ok {
			debug("linter %s disabled", name)
			continue
		}
		parts := strings.SplitN(description, ":", 2)
		command := parts[0]
		pattern := parts[1]

		wg.Add(1)
		go func(name, command, pattern string) {
			concurrency <- true
			executeLinter(issues, name, command, pattern, paths)
			<-concurrency
			wg.Done()
		}(name, command, pattern)
	}

	wg.Wait()
	close(issues)
	for issue := range issues {
		fmt.Printf("%s\n", issue)
	}
	elapsed := time.Now().Sub(start)
	debug("total elapsed time %s", elapsed)
}

func executeLinter(issues chan *Issue, name, command, pattern, paths string) {
	debug("linting with %s: %s", name, command)

	start := time.Now()
	if p, ok := predefinedPatterns[pattern]; ok {
		pattern = p
	}
	re, err := regexp.Compile(pattern)
	kingpin.FatalIfError(err, "invalid pattern for '"+command+"'")

	command = strings.Replace(command, "{paths}", paths, -1)
	debug("executing %s", command)
	cmd := exec.Command("/bin/sh", "-c", command)
	out, err := cmd.CombinedOutput()
	if err != nil {
		if _, ok := err.(*exec.ExitError); !ok {
			debug("warning: %s failed: %s", command, err)
			return
		} else {
			debug("warning: %s returned %s", command, err)
		}
	}
	for _, line := range bytes.Split(out, []byte("\n")) {
		groups := re.FindAllSubmatch(line, -1)
		if groups == nil {
			continue
		}
		issue := &Issue{}
		for i, name := range re.SubexpNames() {
			part := string(groups[0][i])
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

			default:
				kingpin.Fatalf("invalid subgroup %s", name)
			}
		}
		if m, ok := linterMessageOverrideFlag[name]; ok {
			issue.message = strings.Replace(m, "{message}", issue.message, -1)
		}
		if sev, ok := linterSeverityFlag[name]; ok {
			issue.severity = Severity(sev)
		} else {
			issue.severity = "error"
		}
		issues <- issue
	}

	elapsed := time.Now().Sub(start)
	debug("%s linter took %s", name, elapsed)
}
