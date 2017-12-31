// Package engine is the core linting engine of gometalinter.
//
// It takes a set of paths, produces the input for linters (filenames, AST, directories, etc.), calls all linters and
// returns their generated issues.
package engine
