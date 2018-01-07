package main

import (
	"reflect"
	"runtime"
	"testing"

	"github.com/gotestyourself/gotestyourself/assert"
	is "github.com/gotestyourself/gotestyourself/assert/cmp"
)

func TestNewLinterWithCustomLinter(t *testing.T) {
	config := LinterConfig{
		Command: "/usr/bin/custom",
		Pattern: "path",
	}
	linter, err := NewLinter("thename", config)
	assert.NilError(t, err)
	assert.Check(t, is.Equal(functionName(partitionPathsAsDirectories), functionName(linter.LinterConfig.PartitionStrategy)))
	assert.Check(t, is.Equal("(?m:path)", linter.regex.String()))
	assert.Check(t, is.Equal("thename", linter.Name))
	assert.Check(t, is.Equal(config.Command, linter.Command))
}

func TestGetLinterByName(t *testing.T) {
	config := LinterConfig{
		Command:           "maligned",
		Pattern:           "path",
		InstallFrom:       "./install/path",
		PartitionStrategy: partitionPathsAsDirectories,
		IsFast:            true,
	}
	overrideConfig := getLinterByName(config.Command, config)
	assert.Check(t, is.Equal(config.Command, overrideConfig.Command))
	assert.Check(t, is.Equal(config.Pattern, overrideConfig.Pattern))
	assert.Check(t, is.Equal(config.InstallFrom, overrideConfig.InstallFrom))
	assert.Check(t, is.Equal(functionName(config.PartitionStrategy), functionName(overrideConfig.PartitionStrategy)))
	assert.Check(t, is.Equal(config.IsFast, overrideConfig.IsFast))
}

func TestValidateLinters(t *testing.T) {
	originalConfig := *config
	defer func() { config = &originalConfig }()

	config = &Config{
		Enable: []string{"_dummylinter_"},
	}

	err := validateLinters(lintersFromConfig(config), config)
	assert.Assert(t, is.Error(err, "unknown linters: _dummylinter_"))

	config = &Config{
		Enable: defaultEnabled(),
	}
	err = validateLinters(lintersFromConfig(config), config)
	assert.NilError(t, err)
}

func functionName(i interface{}) string {
	return runtime.FuncForPC(reflect.ValueOf(i).Pointer()).Name()
}
