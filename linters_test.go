package main

import (
	"reflect"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewLinterWithCustomLinter(t *testing.T) {
	config := LinterConfig{
		Command: "/usr/bin/custom",
		Pattern: "path",
	}
	linter, err := NewLinter(config)
	require.NoError(t, err)
	assert.NotNil(t, linter.LinterConfig.PartitionStrategy)
}

func TestGetLinterByName(t *testing.T) {
	config := LinterConfig{
		Command:           "aligncheck",
		Pattern:           "path",
		InstallFrom:       "./install/path",
		PartitionStrategy: partitionPathsAsDirectories,
		IsFast:            true,
	}
	overrideConfig := getLinterByName(config.Command, config)
	assert.Equal(t, config.Command, overrideConfig.Command)
	assert.Equal(t, config.Pattern, overrideConfig.Pattern)
	assert.Equal(t, config.InstallFrom, overrideConfig.InstallFrom)
	assert.Equal(t, functionName(config.PartitionStrategy), functionName(overrideConfig.PartitionStrategy))
	assert.Equal(t, config.IsFast, overrideConfig.IsFast)
}

func functionName(i interface{}) string {
	return runtime.FuncForPC(reflect.ValueOf(i).Pointer()).Name()
}
