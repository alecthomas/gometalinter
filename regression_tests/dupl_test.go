package regression_tests

import "testing"

func TestDupl(t *testing.T) {
	t.Parallel()
	source := `package test

func a() {
	if true {
		if false {
			println("yes")
		}
	} else {
		println("what")
	}
}

func b() {
	if true {
		if false {
			println("no")
		}
	} else {
		println("what")
	}
}
`
	expected := Issues{
		{Linter: "dupl", Severity: "warning", Path: "test.go", Line: 13, Col: 0, Message: "duplicate of ./test.go:3-11"},
		{Linter: "dupl", Severity: "warning", Path: "test.go", Line: 3, Col: 0, Message: "duplicate of ./test.go:13-21"},
	}
	ExpectIssues(t, "dupl", source, expected, "--dupl-threshold", "10")
}
