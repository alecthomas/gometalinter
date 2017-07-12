package main

import (
	"runtime"
	"text/template"
	"time"
)

// Config for gometalinter. This can be loaded from a JSON file with --config.
type Config struct { // nolint: aligncheck
	// A map of linter name to "<command>:<pattern>".
	//
	// <command> should always include {path} as the target directory to execute. Globs in <command>
	// are expanded by gometalinter (not by the shell).
	Linters map[string]string

	// The set of linters that should be enabled.
	Enable  []string
	Disable []string

	// A map of linter name to message that is displayed. This is useful when linters display text
	// that is useful only in isolation, such as errcheck which just reports the construct.
	MessageOverride map[string]string
	Severity        map[string]string
	VendoredLinters bool
	Format          string
	Fast            bool
	Install         bool
	Update          bool
	Force           bool
	DownloadOnly    bool
	Debug           bool
	Concurrency     int
	Exclude         []string
	Include         []string
	Skip            []string
	Vendor          bool
	Cyclo           int
	LineLength      int
	MinConfidence   float64
	MinOccurrences  int
	MinConstLength  int
	DuplThreshold   int
	Sort            []string
	Test            bool
	Deadline        time.Duration `json:"-"`
	Errors          bool
	JSON            bool
	Checkstyle      bool
	EnableGC        bool
	Aggregate       bool
	EnableAll       bool

	DeadlineJSONCrutch string `json:"Deadline"`
}

// Configuration defaults.
var (
	vetRe = `^(?:vet:.*?\.go:\s+(?P<path>.*?\.go):(?P<line>\d+):(?P<col>\d+):\s*(?P<message>.*))|(?:(?P<path>.*?\.go):(?P<line>\d+):\s*(?P<message>.*))$`

	// TODO: should be a field on Config struct
	formatTemplate = &template.Template{}
	installMap     = map[string]string{
		"aligncheck":  "github.com/opennota/check/cmd/aligncheck",
		"deadcode":    "github.com/tsenart/deadcode",
		"dupl":        "github.com/mibk/dupl",
		"errcheck":    "github.com/kisielk/errcheck",
		"gas":         "github.com/GoASTScanner/gas",
		"goconst":     "github.com/jgautheron/goconst/cmd/goconst",
		"gocyclo":     "github.com/alecthomas/gocyclo",
		"goimports":   "golang.org/x/tools/cmd/goimports",
		"golint":      "github.com/golang/lint/golint",
		"gosimple":    "honnef.co/go/tools/cmd/gosimple",
		"gotype":      "golang.org/x/tools/cmd/gotype",
		"ineffassign": "github.com/gordonklaus/ineffassign",
		"interfacer":  "github.com/mvdan/interfacer/cmd/interfacer",
		"lll":         "github.com/walle/lll/cmd/lll",
		"megacheck":   "honnef.co/go/tools/cmd/megacheck",
		"misspell":    "github.com/client9/misspell/cmd/misspell",
		"safesql":     "github.com/stripe/safesql",
		"staticcheck": "honnef.co/go/tools/cmd/staticcheck",
		"structcheck": "github.com/opennota/check/cmd/structcheck",
		"unconvert":   "github.com/mdempsky/unconvert",
		"unparam":     "github.com/mvdan/unparam",
		"unused":      "honnef.co/go/tools/cmd/unused",
		"varcheck":    "github.com/opennota/check/cmd/varcheck",
	}
	slowLinters = []string{"structcheck", "varcheck", "errcheck", "aligncheck", "testify", "test", "interfacer", "unconvert", "deadcode", "safesql", "staticcheck", "unparam", "unused", "gosimple", "megacheck"}
	sortKeys    = []string{"none", "path", "line", "column", "severity", "message", "linter"}

	linterTakesFiles = newStringSet("dupl", "gofmt", "goimports", "lll", "misspell")

	linterTakesFilesGroupedByPackage = newStringSet("vet", "vetshadow")

	linterTakesPackagePaths = newStringSet(
		"errcheck",
		"aligncheck",
		"errcheck",
		"gosimple",
		"interfacer",
		"megacheck",
		"safesql",
		"staticcheck",
		"structcheck",
		"test",
		"testify",
		"unconvert",
		"unparam",
		"unused",
		"varcheck",
	)

	// Linter definitions.
	linterDefinitions = map[string]string{
		"aligncheck":  `aligncheck:^(?:[^:]+: )?(?P<path>.*?\.go):(?P<line>\d+):(?P<col>\d+):\s*(?P<message>.+)$`,
		"deadcode":    `deadcode:^deadcode: (?P<path>.*?\.go):(?P<line>\d+):(?P<col>\d+):\s*(?P<message>.*)$`,
		"dupl":        `dupl -plumbing -threshold {duplthreshold}:^(?P<path>.*?\.go):(?P<line>\d+)-\d+:\s*(?P<message>.*)$`,
		"errcheck":    `errcheck -abspath:PATH:LINE:COL:MESSAGE`,
		"gas":         `gas -fmt=csv:^(?P<path>.*?\.go),(?P<line>\d+),(?P<message>[^,]+,[^,]+,[^,]+)`,
		"goconst":     `goconst -min-occurrences {min_occurrences} -min-length {min_const_length}:PATH:LINE:COL:MESSAGE`,
		"gocyclo":     `gocyclo -over {mincyclo}:^(?P<cyclo>\d+)\s+\S+\s(?P<function>\S+)\s+(?P<path>.*?\.go):(?P<line>\d+):(\d+)$`,
		"gofmt":       `gofmt -l -s:^(?P<path>.*?\.go)$`,
		"goimports":   `goimports -l:^(?P<path>.*?\.go)$`,
		"golint":      "golint -min_confidence {min_confidence}:PATH:LINE:COL:MESSAGE",
		"gosimple":    "gosimple:PATH:LINE:COL:MESSAGE",
		"gotype":      "gotype -e {tests=-a}:PATH:LINE:COL:MESSAGE",
		"ineffassign": `ineffassign -n:PATH:LINE:COL:MESSAGE`,
		"interfacer":  `interfacer:PATH:LINE:COL:MESSAGE`,
		"lll":         `lll -g -l {maxlinelength}:PATH:LINE:MESSAGE`,
		"megacheck":   "megacheck:PATH:LINE:COL:MESSAGE",
		"misspell":    "misspell -j 1:PATH:LINE:COL:MESSAGE",
		"safesql":     `safesql:^- (?P<path>.*?\.go):(?P<line>\d+):(?P<col>\d+)$`,
		"staticcheck": "staticcheck:PATH:LINE:COL:MESSAGE",
		"structcheck": `structcheck {tests=-t}:^(?:[^:]+: )?(?P<path>.*?\.go):(?P<line>\d+):(?P<col>\d+):\s*(?P<message>.+)$`,
		"test":        `go test:^--- FAIL: .*$\s+(?P<path>.*?\.go):(?P<line>\d+): (?P<message>.*)$`,
		"testify":     `go test:Location:\s+(?P<path>.*?\.go):(?P<line>\d+)$\s+Error:\s+(?P<message>[^\n]+)`,
		"unconvert":   "unconvert:PATH:LINE:COL:MESSAGE",
		"unparam":     `unparam:PATH:LINE:COL:MESSAGE`,
		"unused":      `unused:PATH:LINE:COL:MESSAGE`,
		"varcheck":    `varcheck:^(?:[^:]+: )?(?P<path>.*?\.go):(?P<line>\d+):(?P<col>\d+):\s*(?P<message>.*)$`,
		"vet":         `go tool vet:` + vetRe,
		"vetshadow":   `go tool vet --shadow:` + vetRe,
	}

	config = &Config{
		Format: "{{.Path}}:{{.Line}}:{{if .Col}}{{.Col}}{{end}}:{{.Severity}}: {{.Message}} ({{.Linter}})",

		Severity: map[string]string{
			"gotype":  "error",
			"test":    "error",
			"testify": "error",
			"vet":     "error",
		},
		MessageOverride: map[string]string{
			"errcheck":    "error return value not checked ({message})",
			"gocyclo":     "cyclomatic complexity {cyclo} of function {function}() is high (> {mincyclo})",
			"gofmt":       "file is not gofmted with -s",
			"goimports":   "file is not goimported",
			"safesql":     "potentially unsafe SQL statement",
			"structcheck": "unused struct field {message}",
			"unparam":     "parameter {message}",
			"varcheck":    "unused variable or constant {message}",
		},
		Enable: []string{
			"aligncheck",
			"deadcode",
			"errcheck",
			"gas",
			"goconst",
			"gocyclo",
			"golint",
			"gotype",
			"ineffassign",
			"interfacer",
			"megacheck",
			"structcheck",
			"unconvert",
			"varcheck",
			"vet",
			"vetshadow",
		},
		VendoredLinters: true,
		Concurrency:     runtime.NumCPU(),
		Cyclo:           10,
		LineLength:      80,
		MinConfidence:   0.8,
		MinOccurrences:  3,
		MinConstLength:  3,
		DuplThreshold:   50,
		Sort:            []string{"none"},
		Deadline:        time.Second * 30,
	}
)
