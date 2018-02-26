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

	"github.com/gotestyourself/gotestyourself/assert"
	is "github.com/gotestyourself/gotestyourself/assert/cmp"
	"github.com/gotestyourself/gotestyourself/fs"
)

type Issue struct {
	Linter   string `json:"linter"`
	Severity string `json:"severity"`
	Path     string `json:"path"`
	Line     int    `json:"line"`
	Col      int    `json:"col"`
	Message  string `json:"message"`
}

func (i *Issue) String() string {
	col := ""
	if i.Col != 0 {
		col = fmt.Sprintf("%d", i.Col)
	}
	return fmt.Sprintf("%s:%d:%s:%s: %s (%s)", strings.TrimSpace(i.Path), i.Line, col, i.Severity, strings.TrimSpace(i.Message), i.Linter)
}

type Issues []Issue

// ExpectIssues runs gometalinter and expects it to generate exactly the
// issues provided.
func ExpectIssues(t *testing.T, linter string, source string, expected Issues, extraFlags ...string) {
	// Write source to temporary directory.
	dir, err := ioutil.TempDir(".", "gometalinter-")
	assert.NilError(t, err)
	defer os.RemoveAll(dir)

	testFile := filepath.Join(dir, "test.go")
	err = ioutil.WriteFile(testFile, []byte(source), 0644)
	assert.NilError(t, err)

	actual := RunLinter(t, linter, dir, extraFlags...)
	assert.Check(t, is.Compare(expected, actual))
}

// RunLinter runs the gometalinter as a binary against the files at path and
// returns the issues it encountered
func RunLinter(t *testing.T, linter string, path string, extraFlags ...string) Issues {
	binary, cleanup := buildBinary(t)
	defer cleanup()

	args := []string{
		"-d", "--disable-all", "--enable", linter, "--json",
		"--sort=path", "--sort=line", "--sort=column", "--sort=message",
		"./...",
	}
	args = append(args, extraFlags...)
	cmd := exec.Command(binary, args...)
	cmd.Dir = path

	errBuffer := new(bytes.Buffer)
	cmd.Stderr = errBuffer

	output, _ := cmd.Output()

	var actual Issues
	err := json.Unmarshal(output, &actual)
	if !assert.Check(t, is.NilError(err)) {
		fmt.Printf("Stderr: %s\n", errBuffer)
		fmt.Printf("Output: %s\n", output)
		return nil
	}
	return filterIssues(actual, linter, path)
}

func buildBinary(t *testing.T) (string, func()) {
	tmpdir := fs.NewDir(t, "regression-test-binary")
	path := tmpdir.Join("gometalinter")
	cmd := exec.Command("go", "build", "-o", path, "..")
	assert.NilError(t, cmd.Run())
	return path, tmpdir.Remove
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
