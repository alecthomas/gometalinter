package regressiontests

import "testing"

func TestGoSimple(t *testing.T) {
	t.Parallel()
	source := `package test

import "regexp"

func a(ok bool, ch chan bool) {
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
		{Linter: "gosimple", Severity: "warning", Path: "test.go", Line: 10, Col: 2, Message: "should use for range instead of for { select {} }"},
		{Linter: "gosimple", Severity: "warning", Path: "test.go", Line: 16, Col: 5, Message: "should omit comparison to bool constant, can be simplified to ok"},
		{Linter: "gosimple", Severity: "warning", Path: "test.go", Line: 6, Col: 2, Message: "should use a simple channel send/receive instead of select with a single case"},
	}
	ExpectIssues(t, "gosimple", source, expected)
}
