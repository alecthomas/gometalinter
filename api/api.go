package api

import (
	"go/ast"
	"go/token"
)

type LinterFactory func() Linter

// Linter base interface.
//
// Concrete implementations should implement this interface AND one of the interfaces below.
type Linter interface {
	Name() string
	// Return configuration struct (or nil) for this linter.
	//
	// The TOML configuration section for this linter will be will be deserialised into this value.
	Config() interface{}
}

// DirectoryLinter lints by directory.
type DirectoryLinter interface {
	Linter
	// LintDirectories lints a set of directories.
	LintDirectories(dirs []string) ([]*Issue, error)
}

// PackageLinter lints by package.
//
// The lint runner will attempt to resolve all paths to packages relative to $GOPATH.
type PackageLinter interface {
	Linter
	// LintPackage lints a set of packages.
	LintPackage(packages []string) ([]*Issue, error)
}

// FileLinter lints individual files.
type FileLinter interface {
	Linter
	// LintFiles lints a set of files grouped by directory.
	//
	// For linters that can lint individual files, simply flatten the map of slices.
	LintFiles(files map[string][]string) ([]*Issue, error)
}

// ASTLinter is a Linter that only needs an AST to lint.
type ASTLinter interface {
	Linter
	LintAST(fset *token.FileSet, files map[string]*ast.File) ([]*Issue, error)
}
