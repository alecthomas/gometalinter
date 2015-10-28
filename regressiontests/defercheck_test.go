package regressiontests

import "testing"

func TestDefercheck(t *testing.T) {
	t.Parallel()
	source := `package test

func test() {
	r, _ := os.Open("test")
	defer r.Close()
	defer r.Close()
}
`
	expected := Issues{
		{Linter: "defercheck", Severity: "error", Path: "test.go", Line: 6, Col: 2, Message: "Repeating defer r.Close() inside function test"},
	}
	ExpectIssues(t, "defercheck", source, expected)
}
