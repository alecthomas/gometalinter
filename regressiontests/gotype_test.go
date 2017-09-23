package regressiontests

import (
	"testing"

	"fmt"

	"github.com/gotestyourself/gotestyourself/fs"
	"github.com/stretchr/testify/assert"
)

func TestGoType(t *testing.T) {
	t.Parallel()

	dir := fs.NewDir(t, "test-gotype",
		fs.WithFile("file.go", fileContent("root")),
		fs.WithDir("sub",
			fs.WithFile("file.go", fileContent("sub"))))
	defer dir.Remove()

	expected := Issues{
		{Linter: "gotype", Severity: "error", Path: "file.go", Line: 4, Col: 6, Message: "foo declared but not used"},
		{Linter: "gotype", Severity: "error", Path: "sub/file.go", Line: 4, Col: 6, Message: "foo declared but not used"},
	}
	actual := RunLinter(t, "gotype", dir.Path())
	assert.Equal(t, expected, actual)
}

func fileContent(pkg string) string {
	return fmt.Sprintf(`package %s

func badFunction() {
	var foo string
}
	`, pkg)
}
