package regressiontests

import "testing"

func TestStaticCheck(t *testing.T) {
	t.Parallel()
	source := `package test

import "regexp"

var v = regexp.MustCompile("*")

func f(ch chan bool) {
	var ok bool
	select {
	case <- ch:
	}

	for {
		select {
		case <- ch:
		}
	}

	if ok == true {
	}
}
`
	expected := Issues{
		{Linter: "staticcheck", Severity: "warning", Path: "test.go", Line: 5, Col: 5, Message: "var v is unused (U1000)"},
		{Linter: "staticcheck", Severity: "warning", Path: "test.go", Line: 5, Col: 27, Message: "error parsing regexp: missing argument to repetition operator: `*` (SA1000)"},
		{Linter: "staticcheck", Severity: "warning", Path: "test.go", Line: 7, Col: 6, Message: "func f is unused (U1000)"},
		{Linter: "staticcheck", Severity: "warning", Path: "test.go", Line: 9, Col: 2, Message: "should use a simple channel send/receive instead of select with a single case (S1000)"},
		{Linter: "staticcheck", Severity: "warning", Path: "test.go", Line: 13, Col: 2, Message: "should use for range instead of for { select {} } (S1000)"},
		{Linter: "staticcheck", Severity: "warning", Path: "test.go", Line: 19, Col: 2, Message: "empty branch (SA9003)"},
		{Linter: "staticcheck", Severity: "warning", Path: "test.go", Line: 19, Col: 5, Message: "should omit comparison to bool constant, can be simplified to ok (S1002)"},
	}
	ExpectIssues(t, "staticcheck", source, expected)
}
