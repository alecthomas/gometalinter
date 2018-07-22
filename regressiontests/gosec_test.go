package regressiontests

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"gotest.tools/fs"
)

func TestGosec(t *testing.T) {
	t.Parallel()
	dir := fs.NewDir(t, "test-gosec",
		fs.WithFile("file.go", gosecFileErrorUnhandled("root")),
		fs.WithDir("sub",
			fs.WithFile("file.go", gosecFileErrorUnhandled("sub"))))
	defer dir.Remove()
	expected := Issues{
		{Linter: "gosec", Severity: "warning", Path: "file.go", Line: 3, Col: 0, Message: "Errors unhandled.,LOW,HIGH"},
		{Linter: "gosec", Severity: "warning", Path: "sub/file.go", Line: 3, Col: 0, Message: "Errors unhandled.,LOW,HIGH"},
	}
	actual := RunLinter(t, "gosec", dir.Path())
	assert.Equal(t, expected, actual)
}

func gosecFileErrorUnhandled(pkg string) string {
	return fmt.Sprintf(`package %s
	func badFunction() string {
		u, _ := ErrorHandle()
		return u
	}
	
	func ErrorHandle() (u string, err error) {
		return u
	}
	`, pkg)
}
