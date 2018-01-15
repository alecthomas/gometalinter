package config

import (
	"fmt"
	"io"
	"os"
	"regexp"
	"runtime"
	"strings"
	"text/template"
	"time"

	"github.com/BurntSushi/toml"

	"github.com/alecthomas/gometalinter/api"
	"github.com/alecthomas/gometalinter/linters"
)

// DefaultIssueFormat used to print an issue.
var DefaultIssueFormat = &Template{template.Must(template.New("output").
	Parse("{{.Path}}:{{.Line}}:{{if .Col}}{{.Col}}{{end}}:{{.Severity}}: {{.Message}} ({{.Linter}})"))}

type Duration time.Duration

func (d *Duration) UnmarshalText(text []byte) error {
	duration, err := time.ParseDuration(string(text))
	*d = Duration(duration)
	return err
}

var predefinedPatterns = map[string]string{
	"PATH:LINE:COL:MESSAGE": `^(?P<path>.*?\.go):(?P<line>\d+):(?P<col>\d+):\s*(?P<message>.*)$`,
	"PATH:LINE:MESSAGE":     `^(?P<path>.*?\.go):(?P<line>\d+):\s*(?P<message>.*)$`,
}

type Regexp struct {
	*regexp.Regexp
}

func (r *Regexp) UnmarshalText(data []byte) (err error) {
	text := string(data)
	if replace, ok := predefinedPatterns[text]; ok {
		text = replace
	}
	r.Regexp, err = regexp.Compile(text)
	return
}

type OutputFormat int

const (
	OutputText OutputFormat = iota
	OutputCheckstyle
	OutputJSON
)

func (o *OutputFormat) UnmarshalText(text []byte) error {
	switch string(text) {
	case "text":
		*o = OutputText
	case "checkstyle":
		*o = OutputCheckstyle
	case "json":
		*o = OutputJSON
	default:
		return fmt.Errorf("invalid output format %q", string(text))
	}
	return nil
}

type Template struct {
	*template.Template
}

func (t *Template) UnmarshalText(text []byte) (err error) {
	t.Template, err = template.New("output").Parse(string(text))
	return err
}

// PartitionStrategy is the directory/file partitioning strategy for external linters.
type PartitionStrategy int

const (
	PartitionByDirectories PartitionStrategy = iota
	PartitionByFiles
	PartitionByPackages
	PartitionByFilesByPackage
	PartitionBySingleDirectory
)

func (p PartitionStrategy) String() string {
	switch p {
	case PartitionByDirectories:
		return "directories"
	case PartitionByFiles:
		return "files"
	case PartitionByPackages:
		return "packages"
	case PartitionByFilesByPackage:
		return "files-by-package"
	case PartitionBySingleDirectory:
		return "single-directory"
	default:
		panic("unknown partition strategy")
	}
}

func (p *PartitionStrategy) UnmarshalText(text []byte) error {
	switch string(text) {
	case "directories":
		*p = PartitionByDirectories
	case "files":
		*p = PartitionByFiles
	case "packages":
		*p = PartitionByPackages
	case "files-by-package":
		*p = PartitionByFilesByPackage
	case "single-directory":
		*p = PartitionBySingleDirectory
	default:
		return fmt.Errorf("unknown partition strategy %q", string(text))
	}
	return nil
}

// ExternalLinterDefinition defines how an external linter is to be executed.
//
// External linters are external commands executed by gometalinter.
type ExternalLinterDefinition struct {
	// Name of linter.
	Name string `toml:"name"`
	// Go package to install linter command from.
	InstallFrom string `toml:"install_from"`
	// Command to run the linter. Linter configuration variables may be referenced in the template.
	Command *Template `toml:"command"`
	// Regex used to match lines from the linter's output.
	Pattern Regexp `toml:"pattern"`
	// Partitioning strategy used by this linter.
	PartitionStrategy PartitionStrategy `toml:"partition"`
	// If true this linter will be enabled when fast mode is used.
	IsFast bool `toml:"is_fast"`
	// Disable this linter by default.
	Disable bool `toml:"disable"`
	// Override the default message from the linter with this text template.
	//
	// Linter configuration variables and named regex captures from the line
	// pattern may be referenced in this template.
	MessageOverride *Template `toml:"message_override"`
	// Severity of the linter if messages do not contain a severity. Defaults to Warning.
	Severity api.Severity `toml:"severity"`
}

// Config for gometalinter.
//
// This can be loaded from a TOML file with --config.
type Config struct { // nolint: maligned
	// Formatting string for text output.
	Format *Template `toml:"format"`
	// Set maximum number of linters to run in parallel.
	Concurrency int `toml:"concurrency"`
	// Regex matching linter issue messages to exclude from output.
	Exclude []Regexp `toml:"exclude"`
	// Override excludes.
	Include []Regexp `toml:"include"`
	// Skip directories with these names.
	SkipDirs []string `toml:"skip_dirs"`
	// Sort order (defaults to no sorting): path, line, column, severity, message, linter
	Sort []string `toml:"sort"`
	// Total deadline before terminating linting.
	Deadline Duration `toml:"deadline"`
	// Only show errors.
	Errors bool `toml:"errors"`
	// Type of output to generate: text (default), checkstyle, json
	Output OutputFormat `toml:"output"`
	// Aggregate identical issues from multiple linters into one.
	Aggregate bool `toml:"aggregate"`
	// Warn if a nolint directive did not suppress any issues.
	WarnUnmatchedDirective bool `toml:"warn_unmatched_directive"`

	// Only run "fast" linters.
	Fast bool `toml:"fast"`
	// Linters to run.
	//
	// The default is to run all linters.
	Enabled []string `toml:"enabled"`

	// Per-linter configuration sections.
	//
	// Each linter can have its own configuration in a section of the form [linter.<linter>].
	Linters map[string]toml.Primitive `toml:"linter"`

	// Define an external linter.
	//
	// Note that this just defines the linter. Configuration for the linter, such as extra variables, etc. is
	// specified in the corresponding [linter.<linter>] section.
	Define map[string]ExternalLinterDefinition `toml:"define"`

	md toml.MetaData
}

// UnmarshalLinterConfig unmarshals a [linter.<linter>] section from the config into the given struct.
func (c *Config) UnmarshalLinterConfig(linter string, v interface{}) error {
	return c.md.PrimitiveDecode(c.Linters[linter], v)
}

// Read configuration from a reader.
func Read(r io.Reader) (*Config, error) {
	config := &Config{
		Format:      DefaultIssueFormat,
		Concurrency: runtime.NumCPU(),
		Sort:        []string{"none"},
		Deadline:    Duration(time.Second * 30),
	}
	// This is effectively a constant.
	for name := range linters.Linters {
		config.Enabled = append(config.Enabled, name)
	}
	md, err := toml.DecodeReader(r, config)
	if err != nil {
		return nil, err
	}
	if len(md.Undecoded()) > 0 {
		keys := []string{}
		// Ignore linter keys.
		for _, key := range md.Undecoded() {
			if !strings.HasPrefix(key.String(), "linter.") {
				keys = append(keys, key.String())
			}
		}
		if len(keys) > 0 {
			return nil, fmt.Errorf("unknown keys %s", strings.Join(keys, ","))
		}
	}
	config.md = md
	return config, err
}

// ReadString reads configuration from a string.
func ReadString(s string) (*Config, error) {
	return Read(strings.NewReader(s))
}

// ReadFile reads configuration from a filename.
func ReadFile(filename string) (*Config, error) {
	r, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer r.Close()
	return Read(r)
}
