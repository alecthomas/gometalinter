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
	// For linters that can lint individual files, simply flatten the slice of slices.
	LintFiles(files [][]string) ([]*Issue, error)
}

// ASTLinter is a Linter that only needs an AST to lint.
type ASTLinter interface {
	Linter
	LintAST(fset *token.FileSet, files []*ast.File) ([]*Issue, error)
}
