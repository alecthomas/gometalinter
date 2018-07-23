package regressiontests

import (
	"fmt"
	"go/build"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGosec(t *testing.T) {
	t.Parallel()
	gopath := os.Getenv("GOPATH")
	if gopath == "" {
		gopath = build.Default.GOPATH
	}
	dirPath := filepath.Join(gopath, "src/goreleasegosectest")
	err := os.MkdirAll(dirPath, 0755)
	assert.NoError(t, err)
	defer os.RemoveAll(dirPath)
	err = ioutil.WriteFile(filepath.Join(dirPath, "file.go"), gosecFileErrorUnhandled("goreleasegosectest"), 0644)
	assert.NoError(t, err)
	subDirPath := filepath.Join(dirPath, "sub")
	err = os.MkdirAll(subDirPath, 0755)
	assert.NoError(t, err)
	err = ioutil.WriteFile(filepath.Join(subDirPath, "file.go"), gosecFileErrorUnhandled("sub"), 0644)
	assert.NoError(t, err)

	expected := Issues{
		{Linter: "gosec", Severity: "warning", Path: "file.go", Line: 3, Col: 0, Message: "Errors unhandled.,LOW,HIGH"},
		{Linter: "gosec", Severity: "warning", Path: "sub/file.go", Line: 3, Col: 0, Message: "Errors unhandled.,LOW,HIGH"},
	}
	actual := RunLinter(t, "gosec", dirPath)
	assert.Equal(t, expected, actual)
}

func gosecFileErrorUnhandled(pkg string) []byte {
	return []byte(fmt.Sprintf(`package %s
	func badFunction() string {
		u, _ := ErrorHandle()
		return u
	}
	
	func ErrorHandle() (u string, err error) {
		return u
	}
	`, pkg))
}
