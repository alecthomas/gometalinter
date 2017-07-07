package main

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParitionToMaxSize(t *testing.T) {
	cmdArgs := []string{"/usr/bin/foo", "-c"}
	paths := []string{"one", "two", "three", "four"}

	parts := partitionToMaxSize(cmdArgs, paths, 24)
	expected := [][]string{
		append(cmdArgs, "one", "two"),
		append(cmdArgs, "three"),
		append(cmdArgs, "four"),
	}
	assert.Equal(t, expected, parts)
}

func TestPartitionToPackageFileGlobs(t *testing.T) {
	tmpdir, err := ioutil.TempDir("", "test-expand-paths")
	require.NoError(t, err)
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

	parts := partitionToPackageFileGlobs(cmdArgs, paths)
	expected := [][]string{
		append(cmdArgs, packagePaths(paths[0], "file.go", "other.go")...),
		append(cmdArgs, packagePaths(paths[1], "file.go", "other.go")...),
	}
	assert.Equal(t, expected, parts)
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
	require.NoError(t, err)
	defer os.RemoveAll(tmpdir)

	cmdArgs := []string{"/usr/bin/foo", "-c"}
	paths := []string{filepath.Join(tmpdir, "one"), filepath.Join(tmpdir, "two")}
	parts := partitionToPackageFileGlobs(cmdArgs, paths)
	assert.Len(t, parts, 0)
}

func TestPartitionToMaxArgSizeWithFileGlobsNoFiles(t *testing.T) {
	tmpdir, err := ioutil.TempDir("", "test-expand-paths")
	require.NoError(t, err)
	defer os.RemoveAll(tmpdir)

	cmdArgs := []string{"/usr/bin/foo", "-c"}
	paths := []string{filepath.Join(tmpdir, "one"), filepath.Join(tmpdir, "two")}
	parts := partitionToMaxArgSizeWithFileGlobs(cmdArgs, paths)
	assert.Len(t, parts, 0)
}
