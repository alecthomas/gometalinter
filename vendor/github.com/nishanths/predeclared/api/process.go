package api

import (
	"fmt"
	"go/ast"
	"go/token"
)

type Config struct {
	// Include method names and field names while checking
	Qualified bool
	// List of predeclared identifiers to not report on.
	IgnoredIdents map[string]bool
}

type Issue struct {
	Ident *ast.Ident
	Kind  string
	fset  *token.FileSet
}

func (i *Issue) Pos() token.Position {
	return i.fset.Position(i.Ident.Pos())
}

func (i Issue) String() string {
	pos := i.fset.Position(i.Ident.Pos())
	return fmt.Sprintf("%s: %s %q has same name as predeclared identifier", pos, i.Kind, i.Ident.Name)
}

func ProcessFile(config *Config, fset *token.FileSet, file *ast.File) []Issue { // nolint: gocyclo
	var issues []Issue

	maybeAdd := func(x *ast.Ident, kind string) {
		if !config.IgnoredIdents[x.Name] && isPredeclaredIdent(x.Name) {
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
			if config.Qualified && x.Fields != nil {
				for _, field := range x.Fields.List {
					for _, name := range field.Names {
						maybeAdd(name, "field")
					}
				}
			}
			return true
		case *ast.InterfaceType:
			if config.Qualified && x.Methods != nil {
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
				if config.Qualified {
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
