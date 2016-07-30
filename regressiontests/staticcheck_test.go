package regressiontests

import "testing"

func TestStaticCheck(t *testing.T) {
	t.Parallel()
	source := `package test

import "regexp"

var v = regexp.MustCompile("*")
`
	expected := Issues{
		{Linter: "staticcheck", Severity: "warning", Path: "test.go", Line: 5, Col: 28, Message: "error parsing regexp: missing argument to repetition operator: `*`"},
	}
	ExpectIssues(t, "staticcheck", source, expected)
}
