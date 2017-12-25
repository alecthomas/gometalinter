package regressiontests

import "testing"

func TestPredeclared(t *testing.T) {
	t.Parallel()
	source := `package test
var iota = 1
func recover() interface{} {}
func copy(a, b []int) {}
`
	expected := Issues{
		{Linter: "predeclared", Severity: "warning", Path: "test.go", Line: 2, Col: 5, Message: "variable iota has same name as predeclared identifier"},
		{Linter: "predeclared", Severity: "warning", Path: "test.go", Line: 3, Col: 6, Message: "function recover has same name as predeclared identifier"},
		{Linter: "predeclared", Severity: "warning", Path: "test.go", Line: 4, Col: 6, Message: "function copy has same name as predeclared identifier"},
	}
	ExpectIssues(t, "predeclared", source, expected)
}
