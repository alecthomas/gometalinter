package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

func checkNoInits(rootPath string) ([]string, error) {
	const recursiveSuffix = string(filepath.Separator) + "..."
	recursive := false
	if strings.HasSuffix(rootPath, recursiveSuffix) {
		recursive = true
		rootPath = rootPath[:len(rootPath)-len(recursiveSuffix)]
	}

	messages := []string{}

	err := filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			if !recursive && path != rootPath {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}

		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			return err
		}

		for _, decl := range file.Decls {
			funcDecl, ok := decl.(*ast.FuncDecl)
			if !ok {
				continue
			}
			filename := fset.Position(funcDecl.Pos()).Filename
			line := fset.Position(funcDecl.Pos()).Line
			name := funcDecl.Name.Name
			if name == "init" && funcDecl.Recv.NumFields() == 0 {
				message := fmt.Sprintf("%s:%d %s function", filename, line, name)
				messages = append(messages, message)
			}
		}
		return nil
	})

	return messages, err
}
