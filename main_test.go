package main

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRelativePackagePath(t *testing.T) {
	var testcases = []struct{
		dir string
		expected string
	}{
		{
			dir: "/abs/path",
			expected: "/abs/path",
		},
		{
			dir: ".",
			expected: ".",
		},
		{
			dir: "./foo",
			expected: "./foo",
		},
		{
			dir: "relative/path",
			expected: "./relative/path",
		},
	}

	for _, testcase := range testcases {
		assert.Equal(t, testcase.expected, relativePackagePath(testcase.dir))
	}
}

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
