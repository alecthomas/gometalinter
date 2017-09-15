package regressiontests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/alecthomas/gometalinter/issues"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type Issues []issues.Issue

// ExpectIssues runs gometalinter and expects it to generate exactly the
// issues provided.
func ExpectIssues(t *testing.T, linter string, source string, expected Issues, extraFlags ...string) {
	// Write source to temporary directory.
	dir, err := ioutil.TempDir(".", "gometalinter-")
	require.NoError(t, err)
	defer os.RemoveAll(dir)

	testFile := filepath.Join(dir, "test.go")
	err = ioutil.WriteFile(testFile, []byte(source), 0644)
	require.NoError(t, err)

	actual := RunLinter(t, linter, dir, extraFlags...)
	AssertEqualIssues(t, expected, actual)
}

// RunLinter runs the gometalinter as a binary against the files at path and
// returns the issues it encountered
func RunLinter(t *testing.T, linter string, path string, extraFlags ...string) []issues.Issue {
	binary, cleanup := buildBinary(t)
	defer cleanup()

	args := []string{"-d", "--disable-all", "--enable", linter, "--json", path}
	args = append(args, extraFlags...)
	cmd := exec.Command(binary, args...)

	errBuffer := new(bytes.Buffer)
	cmd.Stderr = errBuffer

	output, _ := cmd.Output()

	var actual []issues.Issue
	err := json.Unmarshal(output, &actual)
	if !assert.NoError(t, err) {
		fmt.Printf("Stderr: %s\n", errBuffer)
		fmt.Printf("Output: %s\n", output)
		return nil
	}
	return filterIssues(actual, linter, path)
}

func buildBinary(t *testing.T) (string, func()) {
	tmpdir, err := ioutil.TempDir("", "regression-test")
	require.NoError(t, err)
	path := filepath.Join(tmpdir, "binary")
	cmd := exec.Command("go", "build", "-o", path, "..")
	require.NoError(t, cmd.Run())
	return path, func() { os.RemoveAll(tmpdir) }
}

func sortIssues(source Issues) {
	order := []string{"path", "line", "column", "severity", "message", "linter"}
	sort.Sort(&sortedIssues{issues: source, order: order})
}

type sortedIssues struct {
	issues []issues.Issue
	order  []string
}

func (s *sortedIssues) Len() int      { return len(s.issues) }
func (s *sortedIssues) Swap(i, j int) { s.issues[i], s.issues[j] = s.issues[j], s.issues[i] }

func (s *sortedIssues) Less(i, j int) bool {
	l, r := s.issues[i], s.issues[j]
	return issues.Compare(l, r, s.order)
}

// filterIssues to just the issues relevant for the current linter and normalize
// the error message by removing the directory part of the path from both Path
// and Message
func filterIssues(issues Issues, linterName string, dir string) Issues {
	filtered := Issues{}
	for _, issue := range issues {
		if issue.Linter == linterName || linterName == "" {
			issue.Path = strings.Replace(issue.Path, dir+string(os.PathSeparator), "", -1)
			issue.Message = strings.Replace(issue.Message, dir+string(os.PathSeparator), "", -1)
			issue.Message = strings.Replace(issue.Message, dir, "", -1)
			filtered = append(filtered, issue)
		}
	}
	return filtered
}

// AssertEqualIssues compares two lists of issues and asserts they are the
// same list, ignoring the order of the list.
func AssertEqualIssues(t assert.TestingT, expected Issues, actual Issues) bool {
	sortIssues(expected)
	sortIssues(actual)
	return assert.Equal(t, expected, actual)
}
