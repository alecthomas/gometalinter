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
	"go/ast"
	"go/doc"
	"go/parser"
	"go/scanner"
	"go/token"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
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
var ignoredIdents map[string]bool

func initIgnoredIdents() {
	for _, s := range strings.Split(*ignore, ",") {
		ident := strings.TrimSpace(s)
		if ident == "" {
			continue
		}
		if !doc.IsPredeclared(ident) {
			log.Printf("ident %q in -ignore is not a predeclared ident", ident)
			os.Exit(2)
		}
		if ignoredIdents == nil {
			ignoredIdents = make(map[string]bool)
		}
		ignoredIdents[ident] = true
	}
}

func main() {
	log.SetFlags(0)
	log.SetPrefix("predeclared: ")

	flag.Usage = usage
	flag.Parse()
	initIgnoredIdents()

	var fset = token.NewFileSet()
	if flag.NArg() == 0 {
		handleFile(fset, true, "<standard input>", os.Stdout) // use the same filename that gofmt uses
	} else {
		for i := 0; i < flag.NArg(); i++ {
			path := flag.Arg(i)
			info, err := os.Stat(path)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				exitCode = 1
			} else if info.IsDir() {
				handleDir(fset, path)
			} else {
				handleFile(fset, false, path, os.Stdout)
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

func handleFile(fset *token.FileSet, stdin bool, filename string, out io.Writer) {
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

	issues := processFile(fset, file)
	if len(issues) == 0 {
		return
	}

	exitCode = 1

	for _, issue := range issues {
		fmt.Fprintf(out, "%s\n", issue)
	}
}

func handleDir(fset *token.FileSet, p string) {
	if err := filepath.Walk(p, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !isGoFile(info) {
			return nil
		}
		handleFile(fset, false, path, os.Stdout)
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

func isIgnoredIdent(name string) bool {
	return ignoredIdents[name]
}

type Issue struct {
	ident *ast.Ident
	kind  string
	fset  *token.FileSet
}

func (i Issue) String() string {
	pos := i.fset.Position(i.ident.Pos())
	return fmt.Sprintf("%s: %s %s has same name as predeclared identifier", pos, i.kind, i.ident.Name)
}

func processFile(fset *token.FileSet, file *ast.File) []Issue {
	var issues []Issue

	maybeAdd := func(x *ast.Ident, kind string) {
		if !isIgnoredIdent(x.Name) && doc.IsPredeclared(x.Name) {
			issues = append(issues, Issue{x, kind, fset})
		}
	}

	seenValueSpecs := make(map[*ast.ValueSpec]bool)

	// TODO: consider deduping package name issues for files in the
	// same directory.
	maybeAdd(file.Name, "package name")

	for _, spec := range file.Imports {
		if spec.Name == nil {
			continue
		}
		maybeAdd(spec.Name, "import name")
	}

	// Handle declarations and fields.
	// https://golang.org/ref/spec#Declarations_and_scope
	ast.Inspect(file, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.GenDecl:
			var kind string
			switch x.Tok {
			case token.CONST:
				kind = "const"
			case token.VAR:
				kind = "variable"
			default:
				return true
			}
			for _, spec := range x.Specs {
				if vspec, ok := spec.(*ast.ValueSpec); ok && !seenValueSpecs[vspec] {
					seenValueSpecs[vspec] = true
					for _, name := range vspec.Names {
						maybeAdd(name, kind)
					}
				}
			}
			return true
		case *ast.TypeSpec:
			maybeAdd(x.Name, "type")
			return true
		case *ast.StructType:
			if *qualified && x.Fields != nil {
				for _, field := range x.Fields.List {
					for _, name := range field.Names {
						maybeAdd(name, "field")
					}
				}
			}
			return true
		case *ast.InterfaceType:
			if *qualified && x.Methods != nil {
				for _, meth := range x.Methods.List {
					for _, name := range meth.Names {
						maybeAdd(name, "method")
					}
				}
			}
			return true
		case *ast.FuncDecl:
			if x.Recv == nil {
				// it's a function
				maybeAdd(x.Name, "function")
			} else {
				// it's a method
				if *qualified {
					maybeAdd(x.Name, "method")
				}
			}
			// add receivers idents
			if x.Recv != nil {
				for _, field := range x.Recv.List {
					for _, name := range field.Names {
						maybeAdd(name, "receiver")
					}
				}
			}
			// Params and Results will be checked in the *ast.FuncType case.
			return true
		case *ast.FuncType:
			// add params idents
			for _, field := range x.Params.List {
				for _, name := range field.Names {
					maybeAdd(name, "param")
				}
			}
			// add returns idents
			if x.Results != nil {
				for _, field := range x.Results.List {
					for _, name := range field.Names {
						maybeAdd(name, "named return")
					}
				}
			}
			return true
		case *ast.LabeledStmt:
			maybeAdd(x.Label, "label")
			return true
		case *ast.AssignStmt:
			// We only care about short variable declarations, which use token.DEFINE.
			if x.Tok == token.DEFINE {
				for _, expr := range x.Lhs {
					if ident, ok := expr.(*ast.Ident); ok {
						maybeAdd(ident, "variable")
					}
				}
			}
			return true
		default:
			return true
		}
	})

	return issues
}
