package regressiontests

import (
	"runtime"
	"strings"
	"testing"

	"gotest.tools/fs"
	"github.com/stretchr/testify/assert"
)

func TestVet(t *testing.T) {
	t.Parallel()

	dir := fs.NewDir(t, "test-vet",
		fs.WithFile("file.go", vetFile("root")),
		fs.WithFile("file_test.go", vetExternalPackageFile("root_test")),
		fs.WithDir("sub",
			fs.WithFile("file.go", vetFile("sub"))),
		fs.WithDir("excluded",
			fs.WithFile("file.go", vetFile("excluded"))))
	defer dir.Remove()

	var expected Issues
	version := runtime.Version()

	switch {
	case strings.HasPrefix(version, "go1.8"), strings.HasPrefix(version, "go1.9"):
		expected = Issues{
			{Linter: "vet", Severity: "error", Path: "file.go", Line: 7, Col: 0, Message: "missing argument for Printf(\"%d\"): format reads arg 1, have only 0 args"},
			{Linter: "vet", Severity: "error", Path: "file.go", Line: 7, Col: 0, Message: "unreachable code"},
			{Linter: "vet", Severity: "error", Path: "file_test.go", Line: 5, Col: 0, Message: "unreachable code"},
			{Linter: "vet", Severity: "error", Path: "sub/file.go", Line: 7, Col: 0, Message: "missing argument for Printf(\"%d\"): format reads arg 1, have only 0 args"},
			{Linter: "vet", Severity: "error", Path: "sub/file.go", Line: 7, Col: 0, Message: "unreachable code"},
		}
	case  strings.HasPrefix(version, "go1.10"):
		expected = Issues{
			{Linter: "vet", Severity: "error", Path: "file.go", Line: 7, Col: 0, Message: "Printf format %d reads arg #1, but call has only 0 args"},
			{Linter: "vet", Severity: "error", Path: "file.go", Line: 7, Col: 0, Message: "unreachable code"},
			{Linter: "vet", Severity: "error", Path: "file_test.go", Line: 5, Col: 0, Message: "unreachable code"},
			{Linter: "vet", Severity: "error", Path: "sub/file.go", Line: 7, Col: 0, Message: "Printf format %d reads arg #1, but call has only 0 args"},
			{Linter: "vet", Severity: "error", Path: "sub/file.go", Line: 7, Col: 0, Message: "unreachable code"},
		}
	default:
		expected = Issues{
			{Linter: "vet", Severity: "error", Path: "file.go", Line: 7, Col: 0, Message: "Printf format %d reads arg #1, but call has 0 args"},
			{Linter: "vet", Severity: "error", Path: "file.go", Line: 7, Col: 0, Message: "unreachable code"},
			{Linter: "vet", Severity: "error", Path: "file_test.go", Line: 5, Col: 0, Message: "unreachable code"},
			{Linter: "vet", Severity: "error", Path: "sub/file.go", Line: 7, Col: 0, Message: "Printf format %d reads arg #1, but call has 0 args"},
			{Linter: "vet", Severity: "error", Path: "sub/file.go", Line: 7, Col: 0, Message: "unreachable code"},
		}
	}

	actual := RunLinter(t, "vet", dir.Path(), "--skip=excluded")
	assert.Equal(t, expected, actual)
}

func vetFile(pkg string) string {
	return `package ` + pkg + `

import "fmt"

func Something() {
	return
	fmt.Printf("%d")
}
`
}

func vetExternalPackageFile(pkg string) string {
	return `package ` + pkg + `

func Example() {
	return
	println("example")
}
`
}
