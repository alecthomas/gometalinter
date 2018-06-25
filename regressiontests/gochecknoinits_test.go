package regressiontests

import "testing"

func TestGochecknoinits(t *testing.T) {
	t.Parallel()
	source := `package test

	var variable = 1

	type S struct {}

	func (s S) init() {}

	func main() {
		init := func() {}
		init()
	}

	func init() {}
`
	expected := Issues{
		{Linter: "gochecknoinits", Severity: "warning", Path: "test.go", Line: 14, Message: "init function"},
	}
	ExpectIssues(t, "gochecknoinits", source, expected)
}
