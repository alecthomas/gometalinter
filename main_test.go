package main

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gotestyourself/gotestyourself/assert"
	is "github.com/gotestyourself/gotestyourself/assert/cmp"
	"gopkg.in/alecthomas/kingpin.v3-unstable"
)

func TestRelativePackagePath(t *testing.T) {
	var testcases = []struct {
		dir      string
		expected string
	}{
		{
			dir:      "/abs/path",
			expected: "/abs/path",
		},
		{
			dir:      ".",
			expected: ".",
		},
		{
			dir:      "./foo",
			expected: "./foo",
		},
		{
			dir:      "relative/path",
			expected: "./relative/path",
		},
	}

	for _, testcase := range testcases {
		assert.Check(t, is.Equal(testcase.expected, relativePackagePath(testcase.dir)))
	}
}

func TestResolvePathsNoPaths(t *testing.T) {
	paths := resolvePaths(nil, nil)
	assert.Check(t, is.Compare([]string{"."}, paths))
}

func TestResolvePathsNoExpands(t *testing.T) {
	// Non-expanded paths should not be filtered by the skip path list
	paths := resolvePaths([]string{".", "foo", "foo/bar"}, []string{"foo/bar"})
	expected := []string{".", "./foo", "./foo/bar"}
	assert.Check(t, is.Compare(expected, paths))
}

func TestResolvePathsWithExpands(t *testing.T) {
	tmpdir, cleanup := setupTempDir(t)
	defer cleanup()

	mkGoFile(t, tmpdir, "file.go")
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
	paths := resolvePaths([]string{"./...", "foo", "duplicate"}, filterPaths)

	expected := []string{
		".",
		"./duplicate",
		"./foo",
		"./include",
		"./include/foo",
	}
	assert.Check(t, is.Compare(expected, paths))
}

func setupTempDir(t *testing.T) (string, func()) {
	tmpdir, err := ioutil.TempDir("", "test-expand-paths")
	assert.NilError(t, err)

	tmpdir, err = filepath.EvalSymlinks(tmpdir)
	assert.NilError(t, err)

	oldwd, err := os.Getwd()
	assert.NilError(t, err)
	assert.NilError(t, os.Chdir(tmpdir))

	return tmpdir, func() {
		os.RemoveAll(tmpdir)
		assert.NilError(t, os.Chdir(oldwd))
	}
}

func mkDir(t *testing.T, paths ...string) {
	fullPath := filepath.Join(paths...)
	assert.NilError(t, os.MkdirAll(fullPath, 0755))
	mkGoFile(t, fullPath, "file.go")
}

func mkFile(t *testing.T, path string, filename string, content string) {
	err := ioutil.WriteFile(filepath.Join(path, filename), []byte(content), 0644)
	assert.NilError(t, err)
}

func mkGoFile(t *testing.T, path string, filename string) {
	mkFile(t, path, filename, "package foo")
}

func TestPathFilter(t *testing.T) {
	skip := []string{"exclude", "skip.go"}
	pathFilter := newPathFilter(skip)

	var testcases = []struct {
		path     string
		expected bool
	}{
		{path: "exclude", expected: true},
		{path: "something/skip.go", expected: true},
		{path: "skip.go", expected: true},
		{path: ".git", expected: true},
		{path: "_ignore", expected: true},
		{path: "include.go", expected: false},
		{path: ".", expected: false},
		{path: "..", expected: false},
	}

	for _, testcase := range testcases {
		assert.Check(t, is.Equal(testcase.expected, pathFilter(testcase.path)), testcase.path)
	}
}

func TestLoadDefaultConfig(t *testing.T) {
	originalConfig := *config
	defer func() { config = &originalConfig }()

	tmpdir, cleanup := setupTempDir(t)
	defer cleanup()

	mkFile(t, tmpdir, defaultConfigPath, `{"Deadline": "3m"}`)

	app := kingpin.New("test-app", "")
	app.Action(loadDefaultConfig)
	setupFlags(app)

	_, err := app.Parse([]string{})
	assert.NilError(t, err)
	assert.Assert(t, is.Equal(3*time.Minute, config.Deadline.Duration()))
}

func TestNoConfigFlag(t *testing.T) {
	originalConfig := *config
	defer func() { config = &originalConfig }()

	tmpdir, cleanup := setupTempDir(t)
	defer cleanup()

	mkFile(t, tmpdir, defaultConfigPath, `{"Deadline": "3m"}`)

	app := kingpin.New("test-app", "")
	app.Action(loadDefaultConfig)
	setupFlags(app)

	_, err := app.Parse([]string{"--no-config"})
	assert.NilError(t, err)
	assert.Assert(t, is.Equal(30*time.Second, config.Deadline.Duration()))
}

func TestConfigFlagSkipsDefault(t *testing.T) {
	originalConfig := *config
	defer func() { config = &originalConfig }()

	tmpdir, cleanup := setupTempDir(t)
	defer cleanup()

	mkFile(t, tmpdir, defaultConfigPath, `{"Deadline": "3m"}`)
	mkFile(t, tmpdir, "test-config", `{"Fast": true}`)

	app := kingpin.New("test-app", "")
	app.Action(loadDefaultConfig)
	setupFlags(app)

	_, err := app.Parse([]string{"--config", filepath.Join(tmpdir, "test-config")})
	assert.NilError(t, err)
	assert.Assert(t, is.Equal(30*time.Second, config.Deadline.Duration()))
	assert.Assert(t, is.Equal(true, config.Fast))
}

func TestLoadConfigWithDeadline(t *testing.T) {
	originalConfig := *config
	defer func() { config = &originalConfig }()

	tmpfile, err := ioutil.TempFile("", "test-config")
	assert.NilError(t, err)
	defer os.Remove(tmpfile.Name())

	_, err = tmpfile.Write([]byte(`{"Deadline": "3m"}`))
	assert.NilError(t, err)
	assert.NilError(t, tmpfile.Close())

	filename := tmpfile.Name()
	err = loadConfig(nil, &kingpin.ParseElement{Value: &filename}, nil)
	assert.NilError(t, err)

	assert.Assert(t, is.Equal(3*time.Minute, config.Deadline.Duration()))
}

func TestDeadlineFlag(t *testing.T) {
	app := kingpin.New("test-app", "")
	setupFlags(app)
	_, err := app.Parse([]string{"--deadline", "2m"})
	assert.NilError(t, err)
	assert.Assert(t, is.Equal(2*time.Minute, config.Deadline.Duration()))
}

func TestAddPath(t *testing.T) {
	paths := []string{"existing"}
	assert.Check(t, is.Compare(paths, addPath(paths, "existing")))
	expected := []string{"existing", "new"}
	assert.Check(t, is.Compare(expected, addPath(paths, "new")))
}

func TestSetupFlagsLinterFlag(t *testing.T) {
	originalConfig := *config
	defer func() { config = &originalConfig }()

	app := kingpin.New("test-app", "")
	setupFlags(app)
	_, err := app.Parse([]string{"--linter", "a:b:c"})
	assert.NilError(t, err)
	linter, ok := config.Linters["a"]
	assert.Check(t, ok)
	assert.Check(t, is.Equal("b", linter.Command))
	assert.Check(t, is.Equal("c", linter.Pattern))
}

func TestSetupFlagsConfigWithLinterString(t *testing.T) {
	originalConfig := *config
	defer func() { config = &originalConfig }()

	tmpfile, err := ioutil.TempFile("", "test-config")
	assert.NilError(t, err)
	defer os.Remove(tmpfile.Name())

	_, err = tmpfile.Write([]byte(`{"Linters": {"linter": "command:path"} }`))
	assert.NilError(t, err)
	assert.NilError(t, tmpfile.Close())

	app := kingpin.New("test-app", "")
	setupFlags(app)

	_, err = app.Parse([]string{"--config", tmpfile.Name()})
	assert.NilError(t, err)
	linter, ok := config.Linters["linter"]
	assert.Check(t, ok)
	assert.Check(t, is.Equal("command", linter.Command))
	assert.Check(t, is.Equal("path", linter.Pattern))
}

func TestSetupFlagsConfigWithLinterMap(t *testing.T) {
	originalConfig := *config
	defer func() { config = &originalConfig }()

	tmpfile, err := ioutil.TempFile("", "test-config")
	assert.NilError(t, err)
	defer os.Remove(tmpfile.Name())

	_, err = tmpfile.Write([]byte(`{"Linters":
		{"linter":
			{ "Command": "command" }}}`))
	assert.NilError(t, err)
	assert.NilError(t, tmpfile.Close())

	app := kingpin.New("test-app", "")
	setupFlags(app)

	_, err = app.Parse([]string{"--config", tmpfile.Name()})
	assert.NilError(t, err)
	linter, ok := config.Linters["linter"]
	assert.Check(t, ok)
	assert.Check(t, is.Equal("command", linter.Command))
	assert.Check(t, is.Equal("", linter.Pattern))
}

func TestSetupFlagsConfigAndLinterFlag(t *testing.T) {
	originalConfig := *config
	defer func() { config = &originalConfig }()

	tmpfile, err := ioutil.TempFile("", "test-config")
	assert.NilError(t, err)
	defer os.Remove(tmpfile.Name())

	_, err = tmpfile.Write([]byte(`{"Linters":
		{"linter": { "Command": "some-command" }}}`))
	assert.NilError(t, err)
	assert.NilError(t, tmpfile.Close())

	app := kingpin.New("test-app", "")
	setupFlags(app)

	_, err = app.Parse([]string{
		"--config", tmpfile.Name(),
		"--linter", "linter:command:pattern"})
	assert.NilError(t, err)
	linter, ok := config.Linters["linter"]
	assert.Check(t, ok)
	assert.Check(t, is.Equal("command", linter.Command))
	assert.Check(t, is.Equal("pattern", linter.Pattern))
}
