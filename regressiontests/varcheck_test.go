package regressiontests

import "testing"

func TestVarcheck(t *testing.T) {
	t.Parallel()
	source := `package test

var v int
`
	expected := Issues{
		{Linter: "varcheck", Severity: "warning", Path: "test.go", Line: 3, Col: 5, Message: "unused global variable v"},
	}
	ExpectIssues(t, "varcheck", source, expected)
}
