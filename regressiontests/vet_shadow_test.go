package regressiontests

import (
	"runtime"
	"strings"
	"testing"
)

func TestVetShadow(t *testing.T) {
	if strings.HasPrefix(runtime.Version(), "go1.8") {
		t.Skip("go vet does not have a --shadow flag in go1.8")
	}

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

	if version := runtime.Version(); strings.HasPrefix(version, "go1.9") {
		expected = Issues{
			{Linter: "vetshadow", Severity: "warning", Path: "test.go", Line: 7, Col: 0, Message: `declaration of "foo" shadows declaration at test.go:5`},
		}
	}

	ExpectIssues(t, "vetshadow", source, expected)
}
