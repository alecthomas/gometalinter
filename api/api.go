package api

import (
	"go/ast"
	"go/token"
)

type ConfigUnmarshaller func(v interface{}) error

// Linter base interface.
//
// Concrete implementations should implement this interface AND one of the interfaces below.
type Linter interface {
	Name() string
	// Unmarshal TOML config for this linter.
	//
	// Can be as simple as:
	//
	//     return unmarshal(&l.config)
	Config(unmarshal ConfigUnmarshaller) error
}

// ASTLinter is a Linter that only needs an AST to lint.
type ASTLinter interface {
	Linter
	LintAST(fset *token.FileSet, files []*ast.File) ([]*Issue, error)
}
