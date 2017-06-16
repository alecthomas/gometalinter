package main

import (
	"io/ioutil"
	"os"
	"path/filepath"
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

func TestExpandPathsNoPaths(t *testing.T) {
	paths := expandPaths(nil, nil)
	assert.Equal(t, []string{"."}, paths)
}

func TestExpandPathsNoExpands(t *testing.T) {
	// Non-expanded paths should not be filtered by the skip path list
	paths := expandPaths([]string{".", "foo", "foo/bar"}, []string{"foo/bar"})
	expected := []string{".", "./foo", "./foo/bar"}
	assert.Equal(t, expected, paths)
}

func TestExpandPathsWithExpands(t *testing.T) {
	tmpdir, cleanup := setupTempDir(t)
	defer cleanup()

	mkGoFile(t, tmpdir)
	mkDir(t, tmpdir, "exclude")
	mkDir(t, tmpdir, "other", "exclude")
	mkDir(t, tmpdir, "include")
	mkDir(t, tmpdir, "include", "foo")
	mkDir(t, tmpdir, "duplicate")
	mkDir(t, tmpdir, ".exclude")
	mkDir(t, tmpdir, "include", ".exclude")
	mkDir(t, tmpdir, "_exclude")
	mkDir(t, tmpdir, "include", "_exclude")

	filterPaths := []string{"exclude", "other/exclude"}
	paths := expandPaths([]string{"./...", "foo", "duplicate"}, filterPaths)

	expected := []string{
		".",
		"./duplicate",
		"./foo",
		"./include",
		"./include/foo",
	}
	assert.Equal(t, expected, paths)
}

func setupTempDir(t *testing.T) (string, func()) {
	tmpdir, err := ioutil.TempDir("", "test-expand-paths")
	require.NoError(t, err)

	oldwd, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(tmpdir))

	return tmpdir, func() {
		os.RemoveAll(tmpdir)
		require.NoError(t, os.Chdir(oldwd))
	}
}

func mkDir(t *testing.T, paths ...string) {
	fullPath := filepath.Join(paths...)
	require.NoError(t, os.MkdirAll(fullPath, 0755))
	mkGoFile(t, fullPath)
}

func mkGoFile(t *testing.T, path string) {
	content := []byte("package foo")
	err := ioutil.WriteFile(filepath.Join(path, "file.go"), content, 0644)
	require.NoError(t, err)
}

func TestLinterStatePaths(t *testing.T) {
	tmpdir, cleanup := setupTempDir(t)
	defer cleanup()

	mkGoFile(t, tmpdir)
	mkDir(t, tmpdir, "two")
	mkDir(t, tmpdir, "two", "three")

	state := linterState{
		Linter: &Linter{Name: "gofmt"},
		paths: []string{".", "./two", "./two/three"},
	}
	expected := []string{"file.go", "two/file.go", "two/three/file.go"}
	assert.Equal(t, expected, state.Paths())
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
