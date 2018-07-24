package regressiontests

import (
	"fmt"
	"go/build"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"gotest.tools/fs"
)

const projPath = "src/test-gosec"

func TestGosec(t *testing.T) {
	t.Parallel()

	dir := fs.NewDir(t, "test-gosec",
		fs.WithDir("src",
			fs.WithDir("test-gosec",
				fs.WithFile("file.go", gosecFileErrorUnhandled("root")),
				fs.WithDir("sub",
					fs.WithFile("file.go", gosecFileErrorUnhandled("sub"))))))
	defer dir.Remove()

	gopath, err := filepath.EvalSymlinks(dir.Path())
	assert.NoError(t, err)
	err = updateGopath(gopath)
	assert.NoError(t, err, "should update GOPATH with temp dir path")
	defer cleanGopath(gopath)

	expected := Issues{
		{Linter: "gosec", Severity: "warning", Path: "file.go", Line: 3, Col: 0, Message: "Errors unhandled.,LOW,HIGH"},
		{Linter: "gosec", Severity: "warning", Path: "sub/file.go", Line: 3, Col: 0, Message: "Errors unhandled.,LOW,HIGH"},
	}

	actual := RunLinter(t, "gosec", filepath.Join(gopath, projPath))
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

func updateGopath(dir string) error {
	gopath := os.Getenv("GOPATH")
	if gopath == "" {
		gopath = build.Default.GOPATH
	}
	gopath += ":" + dir
	return os.Setenv("GOPATH", gopath)
}

func cleanGopath(dir string) error {
	gopath := os.Getenv("GOPATH")
	gopath = strings.TrimSuffix(gopath, ":"+dir)
	return os.Setenv("GOPATH", gopath)
}
