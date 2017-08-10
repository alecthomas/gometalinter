package main

import (
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

func TestLinterWithOverrideCommand(t *testing.T) {
	command := "gosimple --with --args"
	override := LinterOverrideConfig{
		Command: &command,
	}
	linter := getLinterByOverride("gosimple", override)
	assert.NotNil(t, linter)
	require.Equal(t, command, linter.Command)
}
