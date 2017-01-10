package regressiontests

import "testing"

func TestEndsentence(t *testing.T) {
	t.Parallel()
	source := `package test
// Sentence does not end with punctuation
func Sentence() {}
`
	expected := Issues{
		{Linter: "endsentence", Severity: "warning", Path: "test.go", Line: 3, Col: 1, Message: "Sentence comment should end with period"},
	}
	ExpectIssues(t, "endsentence", source, expected)
}
