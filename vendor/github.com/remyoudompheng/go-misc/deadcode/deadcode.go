package main

import (
	"flag"
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"sort"

	"golang.org/x/tools/go/loader"
)

var exitCode int

var (
	withTestFiles bool
)

func main() {
	flag.BoolVar(&withTestFiles, "test", false, "include test files")
	flag.Parse()
	ctx := &Context{
		withTests: withTestFiles,
	}
	if flag.NArg() == 0 {
		ctx.Load(".")
	} else {
		ctx.Load(flag.Args()...)
	}
	report := ctx.Process()
	for _, obj := range report {
		ctx.errorf(obj.Pos(), "%s is unused", obj.Name())
	}
	os.Exit(exitCode)
}

func fatalf(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, format, args...)
	os.Exit(1)
}

type Context struct {
	cwd       string
	withTests bool

	loader.Config
}

func (ctx *Context) Load(args ...string) {
	for _, arg := range args {
		if ctx.withTests {
			ctx.Config.ImportWithTests(arg)
		} else {
			ctx.Config.Import(arg)
		}
	}
}

// error formats the error to standard error, adding program
// identification and a newline
func (ctx *Context) errorf(pos token.Pos, format string, args ...interface{}) {
	if ctx.cwd == "" {
		ctx.cwd, _ = os.Getwd()
	}
	p := ctx.Config.Fset.Position(pos)
	f, err := filepath.Rel(ctx.cwd, p.Filename)
	if err == nil {
		p.Filename = f
	}
	fmt.Fprintf(os.Stderr, p.String()+": "+format+"\n", args...)
	exitCode = 2
}

func (ctx *Context) Process() []types.Object {
	prog, err := ctx.Config.Load()
	if err != nil {
		fatalf("cannot load packages: %s", err)
	}
	var allUnused []types.Object
	for _, pkg := range prog.Imported {
		unused := doPackage(prog, pkg)
		allUnused = append(allUnused, unused...)
	}
	sort.Sort(objects(allUnused))
	return allUnused
}

func doPackage(prog *loader.Program, pkg *loader.PackageInfo) []types.Object {
	used := make(map[types.Object]bool)
	for _, file := range pkg.Files {
		ast.Inspect(file, func(n ast.Node) bool {
			id, ok := n.(*ast.Ident)
			if !ok {
				return true
			}
			obj := pkg.Info.Uses[id]
			if obj != nil {
				used[obj] = true
			}
			return false
		})
	}

	global := pkg.Pkg.Scope()
	var unused []types.Object
	for _, name := range global.Names() {
		if pkg.Pkg.Name() == "main" && name == "main" {
			continue
		}
		obj := global.Lookup(name)
		if !used[obj] && (pkg.Pkg.Name() == "main" || !ast.IsExported(name)) {
			unused = append(unused, obj)
		}
	}
	return unused
}

type objects []types.Object

func (s objects) Len() int           { return len(s) }
func (s objects) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
func (s objects) Less(i, j int) bool { return s[i].Pos() < s[j].Pos() }
