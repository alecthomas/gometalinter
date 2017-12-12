package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLinterConfigUnmarshalJSON(t *testing.T) {
	source := `{
		"Command": "/bin/custom",
		"PartitionStrategy": "directories"
	}`
	var config StringOrLinterConfig
	err := json.Unmarshal([]byte(source), &config)
	require.NoError(t, err)
	assert.Equal(t, "/bin/custom", config.Command)
	assert.Equal(t, functionName(partitionPathsAsDirectories), functionName(config.PartitionStrategy))
}

func TestFindDefaultConfigFile(t *testing.T) {
	tmpdir, cleanup := setupTempDir(t)
	defer cleanup()

	mkDir(t, tmpdir, "contains")
	mkDir(t, tmpdir, "contains", "foo")
	mkDir(t, tmpdir, "contains", "foo", "bar")
	mkDir(t, tmpdir, "contains", "double")
	mkDir(t, tmpdir, "lacks")

	mkFile(t, filepath.Join(tmpdir, "contains"), defaultConfigPath, "{}")
	mkFile(t, filepath.Join(tmpdir, "contains", "double"), defaultConfigPath, "{}")

	var testcases = []struct {
		dir      string
		expected string
		found    bool
	}{
		{
			dir:      tmpdir,
			expected: "",
			found:    false,
		},
		{
			dir:      filepath.Join(tmpdir, "contains"),
			expected: filepath.Join(tmpdir, "contains", defaultConfigPath),
			found:    true,
		},
		{
			dir:      filepath.Join(tmpdir, "contains", "foo"),
			expected: filepath.Join(tmpdir, "contains", defaultConfigPath),
			found:    true,
		},
		{
			dir:      filepath.Join(tmpdir, "contains", "foo", "bar"),
			expected: filepath.Join(tmpdir, "contains", defaultConfigPath),
			found:    true,
		},
		{
			dir:      filepath.Join(tmpdir, "contains", "double"),
			expected: filepath.Join(tmpdir, "contains", "double", defaultConfigPath),
			found:    true,
		},
		{
			dir:      filepath.Join(tmpdir, "lacks"),
			expected: "",
			found:    false,
		},
	}

	for _, testcase := range testcases {
		require.NoError(t, os.Chdir(testcase.dir))
		configFile, found, err := findDefaultConfigFile()
		assert.Equal(t, testcase.expected, configFile)
		assert.Equal(t, testcase.found, found)
		assert.NoError(t, err)
	}
}
