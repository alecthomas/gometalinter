// Command predeclared prints the names and locations of declarations and
// fields in the given files that have the same name as one of Go's
// predeclared identifiers.
//
// Exit code
//
// The command exits with exit code 1 if an error occurred parsing the given
// files or if it finds predeclared identifiers being overridden. It exits
// with exit code 2 if the command was invoked incorrectly.
//
// Usage
//
// Common usage is:
//
//  predeclared file1.go file2.go dir1 dir2
//
// which prints the list of issues to standard output.
// See 'predeclared -h' for help.
//
// If the '-q' flag isn't specified, the command never reports struct fields,
// interface methods, and method names as issues.
// (These aren't included by default since fields and method are always
// accessed by a qualifier—à la obj.Field—and hence are less likely to cause
// confusion when reading code even if they have the same name as a predeclared
// identifier.)
//
// The '-ignore' flag can be used to specify predeclared identifiers to not
// report issues for. For example, to not report overriding of the predeclared
// identifiers 'new' and 'real', set the flag like so:
//
//  -ignore=new,real
//
// The arguments to the command can either be files or directories. If a directory
// is provided, all Go files in the directory and its subdirectories are checked.
// If no arguments are specified, the command reads from standard input.
package main

import (
	"flag"
	"fmt"
	"go/parser"
	"go/scanner"
	"go/token"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/nishanths/predeclared/api"
)

const help = `Find declarations and fields that override predeclared identifiers.

Usage:
  predeclared [flags] [path ...]

Flags:
  -e	   Report all parse errors, not just the first 10 on different lines
  -ignore  Comma-separated list of predeclared identifiers to not report on
  -q       Include method names and field names while checking
`

func usage() {
	fmt.Fprintf(os.Stderr, help)
	os.Exit(2)
}

var (
	allErrors = flag.Bool("e", false, "")
	ignore    = flag.String("ignore", "", "")
	qualified = flag.Bool("q", false, "")
)

var exitCode = 0

func initConfig() *api.Config {
	config := &api.Config{
		Qualified:     *qualified,
		IgnoredIdents: map[string]bool{},
	}
	for _, s := range strings.Split(*ignore, ",") {
		ident := strings.TrimSpace(s)
		if ident == "" {
			continue
		}
		config.IgnoredIdents[ident] = true
	}
	return config
}

func main() {
	log.SetFlags(0)
	log.SetPrefix("predeclared: ")

	flag.Usage = usage
	flag.Parse()
	config := initConfig()

	var fset = token.NewFileSet()
	if flag.NArg() == 0 {
		handleFile(config, fset, true, "<standard input>", os.Stdout) // use the same filename that gofmt uses
	} else {
		for i := 0; i < flag.NArg(); i++ {
			path := flag.Arg(i)
			info, err := os.Stat(path)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				exitCode = 1
			} else if info.IsDir() {
				handleDir(config, fset, path)
			} else {
				handleFile(config, fset, false, path, os.Stdout)
			}
		}
	}

	os.Exit(exitCode)
}

func parserMode() parser.Mode {
	if *allErrors {
		return parser.ParseComments | parser.AllErrors
	}
	return parser.ParseComments
}

func handleFile(config *api.Config, fset *token.FileSet, stdin bool, filename string, out io.Writer) {
	var src []byte
	var err error
	if stdin {
		src, err = ioutil.ReadAll(os.Stdin)
	} else {
		src, err = ioutil.ReadFile(filename)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		exitCode = 1
		return
	}

	file, err := parser.ParseFile(fset, filename, src, parserMode())
	if err != nil {
		scanner.PrintError(os.Stderr, err)
		exitCode = 1
		return
	}

	issues := api.ProcessFile(config, fset, file)
	if len(issues) == 0 {
		return
	}

	exitCode = 1

	for _, issue := range issues {
		fmt.Fprintf(out, "%s\n", issue)
	}
}

func handleDir(config *api.Config, fset *token.FileSet, p string) {
	if err := filepath.Walk(p, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !isGoFile(info) {
			return nil
		}
		handleFile(config, fset, false, path, os.Stdout)
		return nil
	}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		exitCode = 1
	}
}

func isGoFile(f os.FileInfo) bool {
	// ignore non-Go files
	name := f.Name()
	return !f.IsDir() && !strings.HasPrefix(name, ".") && !strings.HasPrefix(name, "_") && strings.HasSuffix(name, ".go")
}
