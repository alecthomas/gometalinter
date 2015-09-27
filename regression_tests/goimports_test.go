package regression_tests

import "testing"

func TestGoimports(t *testing.T) {
	source := `
package test
func test() {fmt.Println(nil)}
`
	expected := Issues{
		{Linter: "goimports", Severity: "error", Path: "test.go", Line: 1, Col: 0, Message: "missing or unreferenced package imports"},
	}
	ExpectIssues(t, "goimports", source, expected)
}
