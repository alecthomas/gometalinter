package main

import (
	"bytes"
	"fmt"
	"os/exec"
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

func TestExitStatus(t *testing.T) {
	testcases := []struct {
		doc      string
		err      error
		expected int
	}{
		{
			doc:      "no error",
			err:      nil,
			expected: 0,
		},
		{
			doc:      "not exit error",
			err:      fmt.Errorf("other error"),
			expected: panicExitStatus,
		},
		{
			doc:      "exit status 1",
			err:      exec.Command("false").Run(),
			expected: 1,
		},
		{
			doc:      "exit status 2",
			err:      exec.Command("sh", "-c", "exit 2").Run(),
			expected: 2,
		},
	}
	for _, testcase := range testcases {
		assert.Equal(t, testcase.expected, exitStatus(testcase.err), testcase.doc)
	}
}

func TestPreprocessLinterPanicNoPanic(t *testing.T) {
	testcases := []error{
		nil,
		fmt.Errorf("random error"),
		exec.Command("sh", "-c", "exit 2").Run(),
	}

	for _, testcase := range testcases {
		buf := bytes.NewBufferString("output")
		err := preprocessLinterPanic("linter", buf, testcase)
		assert.NoError(t, err)
	}
}

func TestPreprocessLinterPanicWithPanic(t *testing.T) {
	buf := bytes.NewBufferString(`
goroutine 1 [running]:
main.main()
	/go/src/github.com/alecthomas/gometalinter/crashes/main.go:4 +0x64
`)
	processError := exec.Command("sh", "-c", "exit 2").Run()
	err := preprocessLinterPanic("linter", buf, processError)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "linter linter may have panicked")
}
