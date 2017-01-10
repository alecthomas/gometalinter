package linter

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
)

// lintError describes an error that has been detected with the code style.
type lintError struct {
	position token.Position
	name     string
	msg      string
}

// Error implements the error interface.
func (e lintError) Error() string {
	return fmt.Sprintf("%s: %s %s", e.position.String(), e.name, e.msg)
}

// Feedback is a type for keeping the feedback for the files parsed.
type Feedback struct {
	Error error
}

// file is a representation of a source file.
type file struct {
	filename string
	astFile  *ast.File
	fileSet  *token.FileSet
}

// funcDocEndsWithPeriod makes sure that the comment for a func/method
// ends with a period.
func (f *file) funcDocEndsWithPeriod(fn *ast.FuncDecl) *Feedback {
	if fn.Doc == nil {
		return nil
	}

	if !strings.HasSuffix(strings.TrimSpace(fn.Doc.Text()), ".") {
		return &Feedback{
			Error: lintError{
				position: f.fileSet.Position(fn.Pos()),
				name:     fn.Name.Name,
				msg:      "comment should end with period",
			},
		}
	}

	return nil
}

// checkDoc walks nodes and includes the checks for documentation.
func (f *file) checkDoc() []*Feedback {
	var feedback []*Feedback

	f.walk(func(n ast.Node) bool {
		if fn, ok := n.(*ast.FuncDecl); ok {
			fb := f.funcDocEndsWithPeriod(fn)
			if fb != nil {
				feedback = append(feedback, fb)
			}

			return false
		}

		return true
	})

	return feedback
}

func (f *file) walk(fn func(ast.Node) bool) {
	ast.Walk(walker(fn), f.astFile)
}

// walker adapts a function to satisfy the ast.Visitor interface.
// The function return whether the walk should proceed into the node's children.
type walker func(ast.Node) bool

// Visit adheres to the ast.Visitor interface.
func (w walker) Visit(node ast.Node) ast.Visitor {
	if w(node) {
		return w
	}
	return nil
}

// Lint takes filenames to go source files and returns feedback
func Lint(filenames ...string) ([]*Feedback, error) {
	var feedback []*Feedback
	fs := token.NewFileSet()

	for _, name := range filenames {
		node, err := parser.ParseFile(fs, name, nil, parser.ParseComments)
		if err != nil {
			return feedback, err
		}

		f := &file{
			filename: name,
			astFile:  node,
			fileSet:  fs,
		}

		feedback = append(feedback, f.checkDoc()...)
	}

	return feedback, nil
}
