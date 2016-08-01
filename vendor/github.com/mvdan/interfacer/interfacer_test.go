// Copyright (c) 2015, Daniel Martí <mvdan@mvdan.cc>
// See LICENSE for licensing information

package interfacer

import (
	"bytes"
	"flag"
	"fmt"
	"go/ast"
	"go/build"
	"go/parser"
	"go/token"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"github.com/kisielk/gotool"
)

const testdata = "testdata"

var (
	warnsRe  = regexp.MustCompile(`^WARN (.*)\n?$`)
	singleRe = regexp.MustCompile(`([^ ]*) can be ([^ ]*)(,|$)`)
)

func goFiles(t *testing.T, p string) []string {
	if strings.HasSuffix(p, ".go") {
		return []string{p}
	}
	dirs := gotool.ImportPaths([]string{p})
	var paths []string
	for _, dir := range dirs {
		files, err := ioutil.ReadDir(dir)
		if err != nil {
			t.Fatal(err)
		}
		for _, file := range files {
			if file.IsDir() {
				continue
			}
			paths = append(paths, filepath.Join(dir, file.Name()))
		}
	}
	return paths
}

type identVisitor struct {
	fset   *token.FileSet
	idents map[string]token.Position
}

func identKey(pos token.Position, name string) string {
	return fmt.Sprintf("%d %s", pos.Line, name)
}

func (v *identVisitor) Visit(n ast.Node) ast.Visitor {
	switch x := n.(type) {
	case *ast.Ident:
		pos := v.fset.Position(x.Pos())
		v.idents[identKey(pos, x.Name)] = pos
	}
	return v
}

func identPositions(fset *token.FileSet, f *ast.File) map[string]token.Position {
	v := &identVisitor{
		fset:   fset,
		idents: make(map[string]token.Position),
	}
	ast.Walk(v, f)
	return v.idents
}

func wantedWarnings(t *testing.T, p string) []Warn {
	fset := token.NewFileSet()
	var warns []Warn
	for _, path := range goFiles(t, p) {
		src, err := os.Open(path)
		if err != nil {
			t.Fatal(err)
		}
		f, err := parser.ParseFile(fset, path, src, parser.ParseComments)
		src.Close()
		if err != nil {
			t.Fatal(err)
		}
		identPos := identPositions(fset, f)
		for _, group := range f.Comments {
			cm := warnsRe.FindStringSubmatch(group.Text())
			if cm == nil {
				continue
			}
			for _, m := range singleRe.FindAllStringSubmatch(cm[1], -1) {
				vname, tname := m[1], m[2]
				comPos := fset.Position(group.Pos())
				warns = append(warns, Warn{
					Pos:     identPos[identKey(comPos, vname)],
					Name:    vname,
					NewType: tname,
				})
			}
		}
	}
	return warns
}

func doTest(t *testing.T, p string) {
	t.Run(p, func(t *testing.T) {
		warns := wantedWarnings(t, p)
		doTestWarns(t, p, warns, p)
	})
}

func warnsJoin(warns []Warn) string {
	var b bytes.Buffer
	for _, warn := range warns {
		fmt.Fprintln(&b, warn.String())
	}
	return b.String()
}

func doTestWarns(t *testing.T, name string, exp []Warn, args ...string) {
	got, err := CheckArgsList(args)
	if err != nil {
		t.Fatalf("Did not want error in %s:\n%v", name, err)
	}
	if !reflect.DeepEqual(exp, got) {
		t.Fatalf("Output mismatch in %s:\nwant:\n%sgot:\n%s",
			name, warnsJoin(exp), warnsJoin(got))
	}
}

func endNewline(s string) string {
	if strings.HasSuffix(s, "\n") {
		return s
	}
	return s + "\n"
}

func doTestString(t *testing.T, name, exp string, args ...string) {
	var b bytes.Buffer
	switch len(args) {
	case 0:
		args = []string{name}
	case 1:
		if args[0] == "" {
			args = nil
		}
	}
	err := CheckArgsOutput(args, &b, true)
	if err != nil {
		t.Fatalf("Did not want error in %s:\n%v", name, err)
	}
	exp = endNewline(exp)
	got := b.String()
	if exp != got {
		t.Fatalf("Output mismatch in %s:\nExpected:\n%s\nGot:\n%s",
			name, exp, got)
	}
}

func inputPaths(t *testing.T, glob string) []string {
	all, err := filepath.Glob(glob)
	if err != nil {
		t.Fatal(err)
	}
	return all
}

func chdirUndo(t *testing.T, d string) func() {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(d); err != nil {
		t.Fatal(err)
	}
	return func() {
		if err := os.Chdir(wd); err != nil {
			t.Fatal(err)
		}
	}
}

func runFileTests(t *testing.T, paths ...string) {
	defer chdirUndo(t, "files")()
	if len(paths) == 0 {
		paths = inputPaths(t, "*")
	}
	for _, p := range paths {
		doTest(t, p)
	}
}

func runLocalTests(t *testing.T, paths ...string) {
	defer chdirUndo(t, "local")()
	if len(paths) > 0 {
		for _, p := range paths {
			doTest(t, p)
		}
		return
	}
	for _, p := range inputPaths(t, "*") {
		paths = append(paths, "./"+p+"/...")
	}
	for _, p := range paths {
		doTest(t, p)
	}
	// non-recursive
	doTest(t, "./single")
	doTestString(t, "no-args", ".", "")
}

func runNonlocalTests(t *testing.T, paths ...string) {
	defer chdirUndo(t, "src")()
	if len(paths) > 0 {
		for _, p := range paths {
			doTest(t, p)
		}
		return
	}
	paths = inputPaths(t, "*")
	for _, p := range paths {
		doTest(t, p+"/...")
	}
	// local recursive
	doTest(t, "./nested/...")
	// non-recursive
	doTest(t, "single")
	// make sure we don't miss a package's imports
	doTestString(t, "grab-import", "grab-import\ngrab-import/use.go:27:15: s can be def2.Fooer")
	defer chdirUndo(t, "nested/pkg")()
	// relative paths
	doTestString(t, "rel-path", "nested/pkg\nsimple.go:12:17: rc can be Closer", "./...")
}

func TestMain(m *testing.M) {
	flag.Parse()
	if err := os.Chdir(testdata); err != nil {
		panic(err)
	}
	wd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	build.Default.GOPATH = wd
	gotool.DefaultContext.BuildContext.GOPATH = wd
	os.Exit(m.Run())
}

func TestWarns(t *testing.T) {
	runFileTests(t)
	runLocalTests(t)
	runNonlocalTests(t)
}

func doTestError(t *testing.T, name, exp string, args ...string) {
	switch len(args) {
	case 0:
		args = []string{name}
	case 1:
		if args[0] == "" {
			args = nil
		}
	}
	err := CheckArgsOutput(args, ioutil.Discard, false)
	if err == nil {
		t.Fatalf("Wanted error in %s, but none found.", name)
	}
	got := err.Error()
	if exp != got {
		t.Fatalf("Error mismatch in %s:\nExpected:\n%s\nGot:\n%s",
			name, exp, got)
	}
}

func TestErrors(t *testing.T) {
	// non-existent Go file
	doTestError(t, "missing.go", "open missing.go: no such file or directory")
	// local non-existent non-recursive
	doTestError(t, "./missing", "no initial packages were loaded")
	// non-local non-existent non-recursive
	doTestError(t, "missing", "no initial packages were loaded")
	// Mixing Go files and dirs
	doTestError(t, "wrong-args", "named files must be .go files: bar", "foo.go", "bar")
}

func TestExtraArg(t *testing.T) {
	err := CheckArgsOutput([]string{"single", "--", "foo", "bar"}, ioutil.Discard, false)
	got := err.Error()
	want := "unwanted extra args: [foo bar]"
	if got != want {
		t.Fatalf("Error mismatch:\nExpected:\n%s\nGot:\n%s", want, got)
	}
}
