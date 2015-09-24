package regression_tests

import "testing"

func TestIneffassign(t *testing.T) {
	source := `package test

func test() {
	a := 1
}`
	expected := Issues{
		{Linter: "ineffassign", Severity: "warning", Path: "test.go", Line: 4, Col: 2, Message: "a assigned and not used"},
	}
	ExpectIssues(t, "ineffassign", source, expected)
}
