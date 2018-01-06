package engine

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/alecthomas/assert"
	"github.com/alecthomas/gometalinter/api"
	"github.com/alecthomas/gometalinter/config"
)

// ExpectIssues runs gometalinter and expects it to generate exactly the
// issues provided.
func ExpectIssues(t *testing.T, linter string, source string, expected []*api.Issue) {
	// Write source to temporary directory.
	dir, err := ioutil.TempDir(".", "gometalinter-")
	require.NoError(t, err)
	defer os.RemoveAll(dir)

	testFile := filepath.Join(dir, "test.go")
	err = ioutil.WriteFile(testFile, []byte(source), 0644)
	require.NoError(t, err)

	actual := RunLinter(t, linter, dir)
	assert.Equal(t, expected, actual)
}

func RunLinter(t *testing.T, linter string, path string) []*api.Issue {
}

type testLinter struct{}

func (t *testLinter) Name() string        { return "test" }
func (t *testLinter) Config() interface{} { return nil }
func (t *testLinter) LintDirectories(dirs []string) ([]*api.Issue, error) {
}

func TestEngine(t *testing.T) {
	engine, err := New(&config.Config{}, []api.LinterFactory{
		func() api.Linter { return &testLinter{} },
	})
	require.NoError(t, err)
	engine.Lint(targets)
}
