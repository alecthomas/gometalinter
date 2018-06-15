package main

import (
	"reflect"
	"regexp"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewLinterWithCustomLinter(t *testing.T) {
	config := LinterConfig{
		Command: "/usr/bin/custom",
		Pattern: "path",
	}
	linter, err := NewLinter("thename", config)
	require.NoError(t, err)
	assert.Equal(t, functionName(partitionPathsAsDirectories), functionName(linter.LinterConfig.PartitionStrategy))
	assert.Equal(t, "(?m:path)", linter.regex.String())
	assert.Equal(t, "thename", linter.Name)
	assert.Equal(t, config.Command, linter.Command)
}

func TestGetLinterByName(t *testing.T) {
	config := LinterConfig{
		Command:           "maligned",
		Pattern:           "path",
		InstallFrom:       "./install/path",
		PartitionStrategy: partitionPathsAsDirectories,
		IsFast:            true,
	}
	overrideConfig := getLinterByName(config.Command, config)
	assert.Equal(t, config.Command, overrideConfig.Command)
	assert.Equal(t, config.Pattern, overrideConfig.Pattern)
	assert.Equal(t, config.InstallFrom, overrideConfig.InstallFrom)
	assert.Equal(t, functionName(config.PartitionStrategy), functionName(overrideConfig.PartitionStrategy))
	assert.Equal(t, config.IsFast, overrideConfig.IsFast)
}

func TestValidateLinters(t *testing.T) {
	originalConfig := *config
	defer func() { config = &originalConfig }()

	config = &Config{
		Enable: []string{"_dummylinter_"},
	}

	err := validateLinters(lintersFromConfig(config), config)
	require.Error(t, err, "expected unknown linter error for _dummylinter_")

	config = &Config{
		Enable: defaultEnabled(),
	}
	err = validateLinters(lintersFromConfig(config), config)
	require.NoError(t, err)
}

func TestLinter_test(t *testing.T) {
	exampleOutput := `--- FAIL: TestHello (0.00s)
	other_test.go:11: 
			Error Trace:	other_test.go:11
			Error:      	Not equal: 
			            	expected: "This is not"
			            	actual  : "equal to this"
			            	
			            	Diff:
			            	--- Expected
			            	+++ Actual
			            	@@ -1 +1 @@
			            	-This is not
			            	+equal to this
			Test:       	TestHello
	other_test.go:12: this should fail
	other_test.go:13: fail again
	other_test.go:14: last fail
	other_test.go:15:   
	other_test.go:16: 
	require.go:1159: 
			Error Trace:	other_test.go:17
			Error:      	Should be true
			Test:       	TestHello
FAIL
FAIL	test	0.003s`

	pattern := regexp.MustCompile(defaultLinters["test"].Pattern)
	matches := pattern.FindAllStringSubmatch(exampleOutput, -1)
	var errors []map[string]string
	for _, match := range matches {
		m := make(map[string]string)
		for i, name := range pattern.SubexpNames() {
			if i != 0 && name != "" {
				m[name] = string(match[i])
			}
		}
		errors = append(errors, m)
	}

	// Assert expected errors
	assert.Equal(t, "other_test.go", errors[0]["path"])
	assert.Equal(t, "12", errors[0]["line"])
	assert.Equal(t, "this should fail", errors[0]["message"])

	assert.Equal(t, "other_test.go", errors[1]["path"])
	assert.Equal(t, "13", errors[1]["line"])
	assert.Equal(t, "fail again", errors[1]["message"])

	assert.Equal(t, "other_test.go", errors[2]["path"])
	assert.Equal(t, "14", errors[2]["line"])
	assert.Equal(t, "last fail", errors[2]["message"])

	assert.Equal(t, "other_test.go", errors[3]["path"])
	assert.Equal(t, "15", errors[3]["line"])
	assert.Equal(t, "  ", errors[3]["message"])

	// Go metalinter does not support errors without a message as there is little or no output to parse
	// E.g. t.Fail() or t.Error("")
	//  assert.Equal(t, "other_test.go", errors[5]["path"])
	//	assert.Equal(t, "15", errors[5]["line"])
	//	assert.Equal(t, "", errors[5]["message"])
}

func TestLinter_testify(t *testing.T) {
	exampleOutput := `--- FAIL: TestHello (0.00s)
	other_test.go:11: 
			Error Trace:	other_test.go:11
			Error:      	Not equal: 
			            	expected: "This is not"
			            	actual  : "equal to this"
			            	
			            	Diff:
			            	--- Expected
			            	+++ Actual
			            	@@ -1 +1 @@
			            	-This is not
			            	+equal to this
			Test:       	TestHello
	other_test.go:12: this should fail
	other_test.go:13: fail again
	other_test.go:14: last fail
	other_test.go:15:   
	other_test.go:16: 
	require.go:1159: 
			Error Trace:	other_test.go:17
			Error:      	Should be true
			Test:       	TestHello
FAIL
FAIL	test	0.003s`

	pattern := regexp.MustCompile(defaultLinters["testify"].Pattern)
	matches := pattern.FindAllStringSubmatch(exampleOutput, -1)
	var errors []map[string]string
	for _, match := range matches {
		m := make(map[string]string)
		for i, name := range pattern.SubexpNames() {
			if i != 0 && name != "" {
				m[name] = string(match[i])
			}
		}
		errors = append(errors, m)
	}

	// Assert expected errors
	assert.Equal(t, "other_test.go", errors[0]["path"])
	assert.Equal(t, "11", errors[0]["line"])
	assert.Equal(t, "Not equal", errors[0]["message"])

	assert.Equal(t, "other_test.go", errors[1]["path"])
	assert.Equal(t, "17", errors[1]["line"])
	assert.Equal(t, "Should be true", errors[1]["message"])
}

func functionName(i interface{}) string {
	return runtime.FuncForPC(reflect.ValueOf(i).Pointer()).Name()
}
