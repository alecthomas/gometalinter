package engine

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/alecthomas/gometalinter/api"
)

// Contains all of the context required for linting and outputting issues.
type lintContext struct {
	files       map[string][]string
	errors      chan error
	issues      chan *api.Issue
	concurrency chan bool
	context     api.Context
}

// Go runs f concurrently, respecting the concurrency limit of the context.
func (l *lintContext) Go(f func()) {
	l.concurrency <- true
	f()
	<-l.concurrency
}

// Emit issues within the context.
func (l *lintContext) Emit(issues []*api.Issue) {
	for _, i := range issues {
		l.issues <- i
	}
}

func (l *lintContext) Error(err error) {
	l.errors <- err
}

func (l *lintContext) Close() {
	close(l.issues)
	close(l.errors)
}

// Packages converts paths to packages.
func (l *lintContext) Packages() ([]string, error) {
	packages := []string{}
	for dir := range l.files {
		pkg, err := packageNameFromPath(dir)
		if err != nil {
			return nil, err
		}
		packages = append(packages, pkg)
	}
	return packages, nil
}

// Apply issues and error and return true if no error occurred.
func (l *lintContext) Apply(issues []*api.Issue, err error) bool {
	if len(issues) > 0 {
		l.Emit(issues)
	}
	if err != nil {
		l.Error(err)
	}
	return err == nil
}

func packageNameFromPath(path string) (string, error) {
	if !filepath.IsAbs(path) {
		return path, nil
	}
	for _, gopath := range getGoPathList() {
		rel, err := filepath.Rel(filepath.Join(gopath, "src"), path)
		if err != nil {
			continue
		}
		return rel, nil
	}
	return "", fmt.Errorf("%s not in GOPATH", path)
}

// Go 1.8 compatible GOPATH.
func getGoPath() string {
	path := os.Getenv("GOPATH")
	if path == "" {
		user, err := user.Current()
		if err != nil {
			panic(err)
		}
		path = filepath.Join(user.HomeDir, "go")
	}
	return path
}

func getGoPathList() []string {
	return strings.Split(getGoPath(), string(os.PathListSeparator))
}
