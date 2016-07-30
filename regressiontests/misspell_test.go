package regressiontests

import "testing"

func TestMisSpell(t *testing.T) {
	t.Parallel()
	source := `package test
// The langauge is incorrect.
var a = "langauge"
`
	expected := Issues{
		{Linter: "misspell", Severity: "warning", Path: "test.go", Line: 2, Col: 7, Message: "found \"langauge\" a misspelling of \"language\""},
	}
	ExpectIssues(t, "misspell", source, expected)
}
