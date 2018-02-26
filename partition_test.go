package main

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/gotestyourself/gotestyourself/assert"
	is "github.com/gotestyourself/gotestyourself/assert/cmp"
	"github.com/gotestyourself/gotestyourself/env"
)

func TestPartitionToMaxSize(t *testing.T) {
	cmdArgs := []string{"/usr/bin/foo", "-c"}
	paths := []string{"one", "two", "three", "four"}

	parts := partitionToMaxSize(cmdArgs, paths, 24)
	expected := [][]string{
		append(cmdArgs, "one", "two"),
		append(cmdArgs, "three"),
		append(cmdArgs, "four"),
	}
	assert.Check(t, is.Compare(expected, parts))
}

func TestPartitionToPackageFileGlobs(t *testing.T) {
	tmpdir, err := ioutil.TempDir("", "test-expand-paths")
	assert.NilError(t, err)
	defer os.RemoveAll(tmpdir)

	cmdArgs := []string{"/usr/bin/foo", "-c"}
	paths := []string{
		filepath.Join(tmpdir, "one"),
		filepath.Join(tmpdir, "two"),
	}
	for _, dir := range paths {
		mkDir(t, dir)
		mkGoFile(t, dir, "other.go")
	}

	parts, err := partitionPathsAsFilesGroupedByPackage(cmdArgs, paths)
	assert.NilError(t, err)
	expected := [][]string{
		append(cmdArgs, packagePaths(paths[0], "file.go", "other.go")...),
		append(cmdArgs, packagePaths(paths[1], "file.go", "other.go")...),
	}
	assert.Check(t, is.Compare(expected, parts))
}

func packagePaths(dir string, filenames ...string) []string {
	paths := []string{}
	for _, filename := range filenames {
		paths = append(paths, filepath.Join(dir, filename))
	}
	return paths
}

func TestPartitionToPackageFileGlobsNoFiles(t *testing.T) {
	tmpdir, err := ioutil.TempDir("", "test-expand-paths")
	assert.NilError(t, err)
	defer os.RemoveAll(tmpdir)

	cmdArgs := []string{"/usr/bin/foo", "-c"}
	paths := []string{filepath.Join(tmpdir, "one"), filepath.Join(tmpdir, "two")}
	parts, err := partitionPathsAsFilesGroupedByPackage(cmdArgs, paths)
	assert.NilError(t, err)
	assert.Check(t, is.Len(parts, 0))
}

func TestPartitionToMaxArgSizeWithFileGlobsNoFiles(t *testing.T) {
	tmpdir, err := ioutil.TempDir("", "test-expand-paths")
	assert.NilError(t, err)
	defer os.RemoveAll(tmpdir)

	cmdArgs := []string{"/usr/bin/foo", "-c"}
	paths := []string{filepath.Join(tmpdir, "one"), filepath.Join(tmpdir, "two")}
	parts, err := partitionPathsAsFiles(cmdArgs, paths)
	assert.NilError(t, err)
	assert.Check(t, is.Len(parts, 0))
}

func TestPathsToPackagePaths(t *testing.T) {
	root := "/fake/root"
	defer env.Patch(t, "GOPATH", root)()

	packagePaths, err := pathsToPackagePaths([]string{
		filepath.Join(root, "src", "example.com", "foo"),
		"./relative/package",
	})
	assert.NilError(t, err)
	expected := []string{"example.com/foo", "./relative/package"}
	assert.Check(t, is.Compare(expected, packagePaths))
}

func TestPartitionPathsByDirectory(t *testing.T) {
	cmdArgs := []string{"/usr/bin/foo", "-c"}
	paths := []string{"one", "two", "three"}

	parts, err := partitionPathsByDirectory(cmdArgs, paths)
	assert.NilError(t, err)
	expected := [][]string{
		append(cmdArgs, "one"),
		append(cmdArgs, "two"),
		append(cmdArgs, "three"),
	}
	assert.Check(t, is.Compare(expected, parts))

}
