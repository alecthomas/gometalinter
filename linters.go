package main

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/alecthomas/kingpin.v3-unstable"
)

type LinterConfig struct {
	Name              string
	Command           string
	Pattern           string
	InstallFrom       string
	PartitionStrategy partitionStrategy
	IsSlow            bool
	defaultEnabled    bool
}

type Linter struct {
	LinterConfig
	regex *regexp.Regexp
}

// NewLinter returns a new linter from a config
func NewLinter(config LinterConfig) (*Linter, error) {
	if p, ok := predefinedPatterns[config.Pattern]; ok {
		config.Pattern = p
	}
	regex, err := regexp.Compile("(?m:" + config.Pattern + ")")
	if err != nil {
		return nil, err
	}
	return &Linter{
		LinterConfig: config,
		regex:        regex,
	}, nil
}

func (l *Linter) String() string {
	return l.Name
}

var predefinedPatterns = map[string]string{
	"PATH:LINE:COL:MESSAGE": `^(?P<path>.*?\.go):(?P<line>\d+):(?P<col>\d+):\s*(?P<message>.*)$`,
	"PATH:LINE:MESSAGE":     `^(?P<path>.*?\.go):(?P<line>\d+):\s*(?P<message>.*)$`,
}

func getLinterByName(name string, customSpec string) *Linter {
	if customSpec != "" {
		return parseLinterSpec(name, customSpec)
	}
	linter, _ := NewLinter(defaultLinters[name])
	return linter
}

func parseLinterSpec(name string, spec string) *Linter {
	parts := strings.SplitN(spec, ":", 2)
	if len(parts) < 2 {
		kingpin.Fatalf("invalid linter: %q", spec)
	}

	config := defaultLinters[name]
	config.Command, config.Pattern = parts[0], parts[1]

	linter, err := NewLinter(config)
	kingpin.FatalIfError(err, "invalid linter %q", name)
	return linter
}

func makeInstallCommand(linters ...string) []string {
	cmd := []string{"get"}
	if config.VendoredLinters {
		cmd = []string{"install"}
	} else {
		if config.Update {
			cmd = append(cmd, "-u")
		}
		if config.Force {
			cmd = append(cmd, "-f")
		}
		if config.DownloadOnly {
			cmd = append(cmd, "-d")
		}
	}
	if config.Debug {
		cmd = append(cmd, "-v")
	}
	cmd = append(cmd, linters...)
	return cmd
}

func installLintersWithOneCommand(targets []string) error {
	cmd := makeInstallCommand(targets...)
	debug("go %s", strings.Join(cmd, " "))
	c := exec.Command("go", cmd...) // nolint: gas
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Run()
}

func installLintersIndividually(targets []string) {
	failed := []string{}
	for _, target := range targets {
		cmd := makeInstallCommand(target)
		debug("go %s", strings.Join(cmd, " "))
		c := exec.Command("go", cmd...) // nolint: gas
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
	names := make([]string, 0, len(defaultLinters))
	targets := make([]string, 0, len(defaultLinters))
	for name, config := range defaultLinters {
		if config.InstallFrom == "" {
			continue
		}
		names = append(names, name)
		targets = append(targets, config.InstallFrom)
	}
	sort.Strings(names)
	namesStr := strings.Join(names, "\n  ")
	if config.DownloadOnly {
		fmt.Printf("Downloading:\n  %s\n", namesStr)
	} else {
		fmt.Printf("Installing:\n  %s\n", namesStr)
	}
	err := installLintersWithOneCommand(targets)
	if err == nil {
		return
	}
	warning("failed to install one or more linters: %s (installing individually)", err)
	installLintersIndividually(targets)
}

func getDefaultLinters() []*Linter {
	out := []*Linter{}
	for _, config := range defaultLinters {
		linter, err := NewLinter(config)
		kingpin.FatalIfError(err, "invalid linter %q", config.Name)
		out = append(out, linter)
	}
	return out
}

func defaultEnabled() []string {
	enabled := []string{}
	for name, config := range defaultLinters {
		if config.defaultEnabled {
			enabled = append(enabled, name)
		}
	}
	return enabled
}

const vetPattern = `^(?:vet:.*?\.go:\s+(?P<path>.*?\.go):(?P<line>\d+):(?P<col>\d+):\s*(?P<message>.*))|(?:(?P<path>.*?\.go):(?P<line>\d+):\s*(?P<message>.*))$`

var defaultLinters = map[string]LinterConfig{
	"aligncheck": {
		Name:              "aligncheck",
		Command:           "aligncheck",
		Pattern:           `^(?:[^:]+: )?(?P<path>.*?\.go):(?P<line>\d+):(?P<col>\d+):\s*(?P<message>.+)$`,
		InstallFrom:       "github.com/opennota/check/cmd/aligncheck",
		PartitionStrategy: partitionToMaxArgSizeWithPackagePaths,
		IsSlow:            true,
		defaultEnabled:    true,
	},
	"deadcode": {
		Name:              "deadcode",
		Command:           "deadcode",
		Pattern:           `^deadcode: (?P<path>.*?\.go):(?P<line>\d+):(?P<col>\d+):\s*(?P<message>.*)$`,
		InstallFrom:       "github.com/tsenart/deadcode",
		PartitionStrategy: partitionToMaxArgSize,
		IsSlow:            true,
		defaultEnabled:    true,
	},
	"dupl": {
		Name:              "dupl",
		Command:           `dupl -plumbing -threshold {duplthreshold}`,
		Pattern:           `^(?P<path>.*?\.go):(?P<line>\d+)-\d+:\s*(?P<message>.*)$`,
		InstallFrom:       "github.com/mibk/dupl",
		PartitionStrategy: partitionToMaxArgSizeWithFileGlobs,
	},
	"errcheck": {
		Name:              "errcheck",
		Command:           `errcheck -abspath`,
		Pattern:           `PATH:LINE:COL:MESSAGE`,
		InstallFrom:       "github.com/kisielk/errcheck",
		IsSlow:            true,
		PartitionStrategy: partitionToMaxArgSizeWithPackagePaths,
		defaultEnabled:    true,
	},
	"gas": {
		Name:              "gas",
		Command:           `gas -fmt=csv`,
		Pattern:           `^(?P<path>.*?\.go),(?P<line>\d+),(?P<message>[^,]+,[^,]+,[^,]+)`,
		InstallFrom:       "github.com/GoASTScanner/gas",
		PartitionStrategy: partitionToMaxArgSize,
		defaultEnabled:    true,
	},
	"goconst": {
		Name:              "goconst",
		Command:           `goconst -min-occurrences {min_occurrences} -min-length {min_const_length}`,
		Pattern:           `PATH:LINE:COL:MESSAGE`,
		InstallFrom:       "github.com/jgautheron/goconst/cmd/goconst",
		PartitionStrategy: partitionToMaxArgSize,
		defaultEnabled:    true,
	},
	"gocyclo": {
		Name:              "gocyclo",
		Command:           `gocyclo -over {mincyclo}`,
		Pattern:           `^(?P<cyclo>\d+)\s+\S+\s(?P<function>\S+)\s+(?P<path>.*?\.go):(?P<line>\d+):(\d+)$`,
		InstallFrom:       "github.com/alecthomas/gocyclo",
		PartitionStrategy: partitionToMaxArgSize,
		defaultEnabled:    true,
	},
	"gofmt": {
		Name:              "gofmt",
		Command:           `gofmt -l -s`,
		Pattern:           `^(?P<path>.*?\.go)$`,
		PartitionStrategy: partitionToMaxArgSizeWithFileGlobs,
	},
	"goimports": {
		Name:              "goimports",
		Command:           `goimports -l`,
		Pattern:           `^(?P<path>.*?\.go)$`,
		InstallFrom:       "golang.org/x/tools/cmd/goimports",
		PartitionStrategy: partitionToMaxArgSizeWithFileGlobs,
	},
	"golint": {
		Name:              "golint",
		Command:           `golint -min_confidence {min_confidence}`,
		Pattern:           `PATH:LINE:COL:MESSAGE`,
		InstallFrom:       "github.com/golang/lint/golint",
		PartitionStrategy: partitionToMaxArgSize,
		defaultEnabled:    true,
	},
	"gosimple": {
		Name:              "gosimple",
		Command:           `gosimple`,
		Pattern:           `PATH:LINE:COL:MESSAGE`,
		InstallFrom:       "honnef.co/go/tools/cmd/gosimple",
		IsSlow:            true,
		PartitionStrategy: partitionToMaxArgSizeWithPackagePaths,
	},
	"gotype": {
		Name:              "gotype",
		Command:           `gotype -e {tests=-t}`,
		Pattern:           `PATH:LINE:COL:MESSAGE`,
		InstallFrom:       "golang.org/x/tools/cmd/gotype",
		PartitionStrategy: partitionToMaxArgSize,
		defaultEnabled:    true,
	},
	"ineffassign": {
		Name:              "ineffassign",
		Command:           `ineffassign -n`,
		Pattern:           `PATH:LINE:COL:MESSAGE`,
		InstallFrom:       "github.com/gordonklaus/ineffassign",
		PartitionStrategy: partitionToMaxArgSize,
		defaultEnabled:    true,
	},
	"interfacer": {
		Name:              "interfacer",
		Command:           `interfacer`,
		Pattern:           `PATH:LINE:COL:MESSAGE`,
		InstallFrom:       "github.com/mvdan/interfacer/cmd/interfacer",
		IsSlow:            true,
		PartitionStrategy: partitionToMaxArgSizeWithPackagePaths,
		defaultEnabled:    true,
	},
	"lll": {
		Name:              "lll",
		Command:           `lll -g -l {maxlinelength}`,
		Pattern:           `PATH:LINE:MESSAGE`,
		InstallFrom:       "github.com/walle/lll/cmd/lll",
		PartitionStrategy: partitionToMaxArgSizeWithFileGlobs,
	},
	"megacheck": {
		Name:              "megacheck",
		Command:           `megacheck`,
		Pattern:           `PATH:LINE:COL:MESSAGE`,
		InstallFrom:       "honnef.co/go/tools/cmd/megacheck",
		IsSlow:            true,
		PartitionStrategy: partitionToMaxArgSizeWithPackagePaths,
		defaultEnabled:    true,
	},
	"misspell": {
		Name:              "misspell",
		Command:           `misspell -j 1`,
		Pattern:           `PATH:LINE:COL:MESSAGE`,
		InstallFrom:       "github.com/client9/misspell/cmd/misspell",
		PartitionStrategy: partitionToMaxArgSizeWithFileGlobs,
	},
	"safesql": {
		Name:              "safesql",
		Command:           `safesql`,
		Pattern:           `^- (?P<path>.*?\.go):(?P<line>\d+):(?P<col>\d+)$`,
		InstallFrom:       "github.com/stripe/safesql",
		IsSlow:            true,
		PartitionStrategy: partitionToMaxArgSizeWithPackagePaths,
	},
	"staticcheck": {
		Name:              "staticcheck",
		Command:           `staticcheck`,
		Pattern:           `PATH:LINE:COL:MESSAGE`,
		InstallFrom:       "honnef.co/go/tools/cmd/staticcheck",
		IsSlow:            true,
		PartitionStrategy: partitionToMaxArgSizeWithPackagePaths,
	},
	"structcheck": {
		Name:              "structcheck",
		Command:           `structcheck {tests=-t}`,
		Pattern:           `^(?:[^:]+: )?(?P<path>.*?\.go):(?P<line>\d+):(?P<col>\d+):\s*(?P<message>.+)$`,
		InstallFrom:       "github.com/opennota/check/cmd/structcheck",
		IsSlow:            true,
		PartitionStrategy: partitionToMaxArgSizeWithPackagePaths,
		defaultEnabled:    true,
	},
	"test": {
		Name:              "test",
		Command:           `go test`,
		Pattern:           `^--- FAIL: .*$\s+(?P<path>.*?\.go):(?P<line>\d+): (?P<message>.*)$`,
		IsSlow:            true,
		PartitionStrategy: partitionToMaxArgSizeWithPackagePaths,
	},
	"testify": {
		Name:              "testify",
		Command:           `go test`,
		Pattern:           `Location:\s+(?P<path>.*?\.go):(?P<line>\d+)$\s+Error:\s+(?P<message>[^\n]+)`,
		IsSlow:            true,
		PartitionStrategy: partitionToMaxArgSizeWithPackagePaths,
	},
	"unconvert": {
		Name:              "unconvert",
		Command:           `unconvert`,
		Pattern:           `PATH:LINE:COL:MESSAGE`,
		InstallFrom:       "github.com/mdempsky/unconvert",
		IsSlow:            true,
		PartitionStrategy: partitionToMaxArgSizeWithPackagePaths,
		defaultEnabled:    true,
	},
	"unparam": {
		Name:              "unparam",
		Command:           `unparam`,
		Pattern:           `PATH:LINE:COL:MESSAGE`,
		InstallFrom:       "github.com/mvdan/unparam",
		IsSlow:            true,
		PartitionStrategy: partitionToMaxArgSizeWithPackagePaths,
	},
	"unused": {
		Name:              "unused",
		Command:           `unused`,
		Pattern:           `PATH:LINE:COL:MESSAGE`,
		InstallFrom:       "honnef.co/go/tools/cmd/unused",
		IsSlow:            true,
		PartitionStrategy: partitionToMaxArgSizeWithPackagePaths,
	},
	"varcheck": {
		Name:              "varcheck",
		Command:           `varcheck`,
		Pattern:           `^(?:[^:]+: )?(?P<path>.*?\.go):(?P<line>\d+):(?P<col>\d+):\s*(?P<message>.*)$`,
		InstallFrom:       "github.com/opennota/check/cmd/varcheck",
		IsSlow:            true,
		PartitionStrategy: partitionToMaxArgSizeWithPackagePaths,
		defaultEnabled:    true,
	},
	"vet": {
		Name:              "vet",
		Command:           `go tool vet`,
		Pattern:           vetPattern,
		PartitionStrategy: partitionToPackageFileGlobs,
		defaultEnabled:    true,
	},
	"vetshadow": {
		Name:              "vetshadow",
		Command:           `go tool vet --shadow`,
		Pattern:           vetPattern,
		PartitionStrategy: partitionToPackageFileGlobs,
		defaultEnabled:    true,
	},
}
