package regressiontests

import "testing"

func TestVetShadow(t *testing.T) {
	t.Parallel()
	source := `package test

type MyStruct struct {}
func test(mystructs []*MyStruct) *MyStruct {
	var foo *MyStruct
	for _, mystruct := range mystructs {
		foo := mystruct
	}
	return foo
}
`
	expected := Issues{
		{Linter: "vetshadow", Severity: "warning", Path: "test.go", Line: 7, Col: 3, Message: "foo declared and not used"},
	}
	ExpectIssues(t, "vetshadow", source, expected)
}
