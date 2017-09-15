package regressiontests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
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

	// Run gometalinter.
	binary, cleanup := buildBinary(t)
	defer cleanup()
	args := []string{"-d", "--disable-all", "--enable", linter, "--json", dir}
	args = append(args, extraFlags...)
	cmd := exec.Command(binary, args...)
	errBuffer := new(bytes.Buffer)
	cmd.Stderr = errBuffer
	require.NoError(t, err)

	output, _ := cmd.Output()
	var actual Issues
	err = json.Unmarshal(output, &actual)
	if !assert.NoError(t, err) {
		fmt.Printf("Stderr: %s\n", errBuffer)
		fmt.Printf("Output: %s\n", output)
		return
	}

	// Remove output from other linters.
	actualForLinter := filterIssues(actual, linter, testFile)
	sort(expected)
	sort(actualForLinter)

	if !assert.Equal(t, expected, actualForLinter) {
		fmt.Printf("Stderr: %s", errBuffer)
		fmt.Printf("Output: %s", output)
	}
}

func buildBinary(t *testing.T) (string, func()) {
	tmpdir, err := ioutil.TempDir("", "regression-test")
	require.NoError(t, err)
	path := filepath.Join(tmpdir, "binary")
	cmd := exec.Command("go", "build", "-o", path, "..")
	require.NoError(t, cmd.Run())
	return path, func() { os.RemoveAll(tmpdir) }
}

func sort(source Issues) {
	withRef := []*issues.Issue{}
	for _, issue := range source {
		withRef = append(withRef, &issue)
	}
	issues.Sort(withRef, []string{"path", "line", "column", "severity"})
}

func filterIssues(issues Issues, linterName string, filename string) Issues {
	actualForLinter := Issues{}
	for _, issue := range issues {
		if issue.Linter == linterName || linterName == "" {
			// Normalise path.
			issue.Path = "test.go"
			issue.Message = strings.Replace(issue.Message, filename, "test.go", -1)
			actualForLinter = append(actualForLinter, issue)
		}
	}
	return actualForLinter
}
