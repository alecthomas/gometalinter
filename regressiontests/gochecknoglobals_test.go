package regressiontests

import "testing"

func TestGochecknoglobals(t *testing.T) {
	t.Parallel()
	source := `package test

	var _ = 1

	const constant = 2

	var globalVar = 3

	func function() int {
		var localVar = 4
		return localVar
	}
`
	expected := Issues{
		{Linter: "gochecknoglobals", Severity: "warning", Path: "test.go", Line: 7, Message: "globalVar is a global variable"},
	}
	ExpectIssues(t, "gochecknoglobals", source, expected)
}
