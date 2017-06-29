package regressiontests

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
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

func (e Issues) Len() int           { return len(e) }
func (e Issues) Swap(i, j int)      { e[i], e[j] = e[j], e[i] }
func (e Issues) Less(i, j int) bool { return e[i].String() < e[j].String() }

// ExpectIssues runs gometalinter and expects it to generate exactly the
// issues provided.
func ExpectIssues(t *testing.T, linter string, source string, expected Issues, extraFlags ...string) {
	// Write source to temporary directory.
	dir, err := ioutil.TempDir(".", "gometalinter-")
	if !assert.NoError(t, err) {
		return
	}
	defer os.RemoveAll(dir)
	w, err := os.Create(filepath.Join(dir, "test.go"))
	if !assert.NoError(t, err) {
		return
	}
	defer os.Remove(w.Name())
	_, err = w.WriteString(source)
	_ = w.Close()
	if !assert.NoError(t, err) {
		return
	}

	// Run gometalinter.
	args := []string{"go", "run", "../main.go", "../directives.go", "../config.go", "../checkstyle.go", "../aggregate.go", "--disable-all", "--enable", linter, "--json", dir}
	args = append(args, extraFlags...)
	cmd := exec.Command(args[0], args[1:]...)
	if !assert.NoError(t, err) {
		return
	}
	output, _ := cmd.Output()
	var actual Issues
	err = json.Unmarshal(output, &actual)
	if !assert.NoError(t, err) {
		fmt.Printf("Output: %s\n", output)
		return
	}

	// Remove output from other linters.
	actualForLinter := Issues{}
	for _, issue := range actual {
		if issue.Linter == linter || linter == "" {
			// Normalise path.
			issue.Path = "test.go"
			issue.Message = strings.Replace(issue.Message, w.Name(), "test.go", -1)
			issue.Message = strings.Replace(issue.Message, dir, "", -1)
			actualForLinter = append(actualForLinter, issue)
		}
	}
	sort.Sort(expected)
	sort.Sort(actualForLinter)

	assert.Equal(t, expected, actualForLinter)
}
