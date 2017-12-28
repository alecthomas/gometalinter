// Package config reads the gometalinter TOML configuration file.
package config

import (
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/alecthomas/gometalinter/api"
)

type Duration time.Duration

func (d *Duration) UnmarshalText(text []byte) error {
	duration, err := time.ParseDuration(string(text))
	*d = Duration(duration)
	return err
}

type Regexp struct {
	*regexp.Regexp
}

func (r *Regexp) UnmarshalText(text []byte) error {
	re, err := regexp.Compile(string(text))
	r.Regexp = re
	return err
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

// Config for gometalinter.
//
// This can be loaded from a TOML file with --config.
type Config struct { // nolint: maligned
	// Formatting string for text output.
	Format string `toml:"format"`
	// Only run "fast" linters.
	Fast bool `toml:"fast"`
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
	// Enable linting of tests for those linters that support it.
	Test bool `toml:"test"`
	// Total deadline before terminating linting.
	Deadline Duration `toml:"deadline"`
	// Only show errors.
	Errors bool `toml:"errors"`
	// Type of output to generate: text (default), checkstyle, json
	Output OutputFormat `toml:"output"`
	// Aggregate identical issues from multiple linters into one.
	Aggregate bool `toml:"aggregate"`
	// Enable all linters, even default-disabled ones.
	EnableAll bool `toml:"enable_all"`
	// Warn if a nolint directive did not suppress any issues.
	WarnUnmatchedDirective bool `toml:"warn_unmatched_directive"`

	// Per-linter configuration sections.
	//
	// Each linter can have its own configuration in a section of the form [linter.<linter>].
	Linters map[string]toml.Primitive `toml:"linter"`

	md toml.MetaData
}

// UnmarshalLinterConfig unmarshals a [linter.<linter>] section from the config into the given struct.
func (c *Config) UnmarshalLinterConfig(linter string, v interface{}) error {
	return c.md.PrimitiveDecode(c.Linters[linter], v)
}

// Read configuration from a reader.
func Read(r io.Reader) (*Config, error) {
	config := &Config{
		Format: api.DefaultIssueFormat,
	}
	md, err := toml.DecodeReader(r, config)
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
