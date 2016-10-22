package regressiontests

import (
	"reflect"
	"testing"
)

type Empty struct{}

func TestStructcheck(t *testing.T) {
	t.Parallel()
	source := `package test

type test struct {
	unused int
}
`
	pkgName := reflect.TypeOf(Empty{}).PkgPath()
	expected := Issues{
		{Linter: "structcheck", Severity: "warning", Path: "test.go", Line: 4, Col: 2, Message: "unused struct field " + pkgName + "/.test.unused"},
	}
	ExpectIssues(t, "structcheck", source, expected)
}
