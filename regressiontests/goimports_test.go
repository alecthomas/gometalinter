package regressiontests

import "testing"

func TestGoimports(t *testing.T) {
	source := `
package test
func test() {fmt.Println(nil)}
`
	expected := Issues{
		{Linter: "goimports", Severity: "error", Path: "test.go", Line: 1, Col: 0, Message: "file is not goimported"},
	}
	ExpectIssues(t, "goimports", source, expected)
}
