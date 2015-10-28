package regressiontests

import "testing"

func TestGofmt(t *testing.T) {
	t.Parallel()
	source := `
package test
func test() { if nil {} }
`
	expected := Issues{
		{Linter: "gofmt", Severity: "error", Path: "test.go", Line: 1, Col: 0, Message: "file is not gofmted"},
	}
	ExpectIssues(t, "gofmt", source, expected)
}
