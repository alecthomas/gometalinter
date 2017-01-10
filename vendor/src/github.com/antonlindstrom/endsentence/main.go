package main

import (
	"flag"
	"fmt"
	"go/build"
	"os"
	"path/filepath"

	"github.com/antonlindstrom/endsentence/linter"
)

var buildref string

// filesFromPackage imports files from a package.
func filesFromPackage(pkg *build.Package) []string {
	var files []string

	files = append(files, pkg.GoFiles...)
	files = append(files, pkg.CgoFiles...)
	files = append(files, pkg.TestGoFiles...)

	if pkg.Dir != "." {
		for i, f := range files {
			files[i] = filepath.Join(pkg.Dir, f)
		}
	}

	return files
}

func isDir(dirname string) bool {
	fi, err := os.Stat(dirname)
	return err == nil && fi.IsDir()
}

func lint(fs ...string) []*linter.Feedback {
	if len(fs) == 1 && isDir(fs[0]) {
		pkg, err := build.ImportDir(fs[0], 0)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		feedback, err := linter.Lint(filesFromPackage(pkg)...)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		return feedback
	}

	feedback, err := linter.Lint(fs...)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	return feedback
}

func main() {
	var version = flag.Bool("version", false, "Print build version")
	flag.Parse()

	if *version {
		fmt.Printf("endsentence build: %v\n", buildref)
		return
	}

	var feedback []*linter.Feedback

	switch flag.NArg() {
	case 0:
		feedback = lint(".")
	case 1:
		feedback = lint(flag.Arg(0))
	default:
		feedback = lint(flag.Args()...)
	}

	for _, f := range feedback {
		fmt.Fprintln(os.Stderr, f.Error)
	}

	if len(feedback) > 0 {
		os.Exit(1)
	}
}
