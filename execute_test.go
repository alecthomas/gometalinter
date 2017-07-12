package main

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSortedIssues(t *testing.T) {
	actual := []*Issue{
		{Path: "b.go", Line: 5},
		{Path: "a.go", Line: 3},
		{Path: "b.go", Line: 1},
		{Path: "a.go", Line: 1},
	}
	issues := &sortedIssues{
		issues: actual,
		order:  []string{"path", "line"},
	}
	sort.Sort(issues)
	expected := []*Issue{
		{Path: "a.go", Line: 1},
		{Path: "a.go", Line: 3},
		{Path: "b.go", Line: 1},
		{Path: "b.go", Line: 5},
	}
	require.Equal(t, expected, actual)
}

func TestLinterStatePartitions(t *testing.T) {
	noPartitions := func(_ []string, _ []string) ([][]string, error) {
		return nil, nil
	}

	state := &linterState{
		Linter: &Linter{
			Name:              "thelinter",
			partitionStrategy: noPartitions,
			Command:           "go",
		},
	}

	_, err := state.Partitions()
	assert.EqualError(t, err, "thelinter: no files to lint")
}
