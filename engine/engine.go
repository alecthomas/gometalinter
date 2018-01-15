package engine

import (
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/kisielk/gotool"

	"github.com/alecthomas/gometalinter/api"
	"github.com/alecthomas/gometalinter/config"
	. "github.com/alecthomas/gometalinter/util" // nolint: golint
)

type LinterType int

const (
	DirectoryLinterType LinterType = iota
	PackageLinterType
	FileLinterType
	ASTLinterType
)

type Engine struct {
	config        *config.Config
	linters       map[string]api.Linter
	lintersByType map[LinterType][]api.Linter
	sort          []string
	include       *regexp.Regexp
	exclude       *regexp.Regexp
}

// New creates a new linter engine.
func New(conf *config.Config, linters []api.LinterFactory) (*Engine, error) {
	mapping := map[string]api.Linter{}
	lintersByType := map[LinterType][]api.Linter{}
	for _, f := range linters {
		l := f()
		err := conf.UnmarshalLinterConfig(l.Name(), l.Config())
		if err != nil {
			return nil, err
		}
		mapping[l.Name()] = l
		switch l.(type) {
		case api.DirectoryLinter:
			lintersByType[DirectoryLinterType] = append(lintersByType[DirectoryLinterType], l)
		case api.PackageLinter:
			lintersByType[PackageLinterType] = append(lintersByType[PackageLinterType], l)
		case api.FileLinter:
			lintersByType[FileLinterType] = append(lintersByType[FileLinterType], l)
		case api.ASTLinter:
			lintersByType[ASTLinterType] = append(lintersByType[ASTLinterType], l)
		default:
			return nil, fmt.Errorf("linter %q does not implement any concrete linter interfaces", l.Name())
		}
	}
	order := conf.Sort
	// Force sorting by path if checkstyle mode is selected
	if conf.Output == config.OutputCheckstyle {
		order = []string{"path"}
	}
	var include, exclude *regexp.Regexp
	if len(conf.Exclude) > 0 {
		excludes := []string{}
		for _, e := range conf.Exclude {
			excludes = append(excludes, e.String())
		}
		exclude = regexp.MustCompile(strings.Join(excludes, "|"))
	}

	if len(conf.Include) > 0 {
		includes := []string{}
		for _, i := range conf.Exclude {
			includes = append(includes, i.String())
		}
		include = regexp.MustCompile(strings.Join(includes, "|"))
	}
	return &Engine{
		config:        conf,
		sort:          order,
		linters:       mapping,
		lintersByType: lintersByType,
		include:       include,
		exclude:       exclude,
	}, nil
}

// Lint targets.
//
// Targets are in the form accepted by gotool (eg. <dir>, <dir>/..., etc.)
func (e *Engine) Lint(targets []string) (chan *api.Issue, chan error) {
	issues := make(chan *api.Issue, 1)
	errors := make(chan error, 1)

	// Enumerate all files in all directories.
	filesByDir, err := expandDirs(e.resolvePaths(targets))
	if err != nil {
		errors <- err
		close(issues)
		close(errors)
		return issues, errors
	}
	context := &lintContext{
		concurrency: make(chan bool, e.config.Concurrency),
		errors:      errors,
		issues:      issues,
		files:       filesByDir,
	}
	go func() {
		defer context.Close()
		if len(e.lintersByType[DirectoryLinterType]) > 0 {
			e.runDirectoryLinters(context)
		}
		if len(e.lintersByType[PackageLinterType]) > 0 {
			e.runPackageLinters(context)
		}
		if len(e.lintersByType[FileLinterType]) > 0 {
			e.runFileLinters(context)
		}
		if len(e.lintersByType[ASTLinterType]) > 0 {
			e.runASTLinters(context)
		}
	}()
	return issues, errors
}

func (e *Engine) runDirectoryLinters(context *lintContext) {
	dirs := []string{}
	for dir := range context.files {
		dirs = append(dirs, dir)
	}
	for _, linter := range e.lintersByType[DirectoryLinterType] {
		context.Go(func() { context.Apply(linter.(api.DirectoryLinter).LintDirectories(dirs)) })
	}
}

func (e *Engine) runPackageLinters(context *lintContext) {
	pkgs, err := context.Packages()
	if err != nil {
		context.Error(err)
		return
	}
	for _, linter := range e.lintersByType[PackageLinterType] {
		context.Go(func() { context.Apply(linter.(api.PackageLinter).LintPackage(pkgs)) })
	}
}

func (e *Engine) runFileLinters(context *lintContext) {
	for _, linter := range e.lintersByType[PackageLinterType] {
		context.Go(func() { context.Apply(linter.(api.FileLinter).LintFiles(context.files)) })
	}
}

func (e *Engine) runASTLinters(context *lintContext) {
	for dir := range context.files {
		fset := token.NewFileSet()
		pkgs, err := parser.ParseDir(fset, dir, nil, 0)
		if err != nil {
			context.Error(err)
			continue
		}
		for _, pkg := range pkgs {
			for _, linter := range e.lintersByType[ASTLinterType] {
				context.Go(func() { context.Apply(linter.(api.ASTLinter).LintAST(fset, pkg.Files)) })
			}
		}
	}
}

func (e *Engine) resolvePaths(paths []string) []string {
	if len(paths) == 0 {
		return []string{"."}
	}

	skipPath := newPathFilter(e.config.SkipDirs)
	dirs := newStringSet()
	for _, dir := range gotool.ImportPaths(paths) {
		if !skipPath(dir) {
			dirs.add(relativePackagePath(dir))
		}
	}
	out := dirs.asSlice()
	sort.Strings(out)
	for _, d := range out {
		Debug("linting path %s", d)
	}
	return out
}

func newPathFilter(skip []string) func(string) bool {
	filter := map[string]bool{}
	for _, name := range skip {
		filter[name] = true
	}

	return func(path string) bool {
		base := filepath.Base(path)
		if filter[base] || filter[path] {
			return true
		}
		return base != "." && base != ".." && strings.ContainsAny(base[0:1], "_.")
	}
}

func relativePackagePath(dir string) string {
	if filepath.IsAbs(dir) || strings.HasPrefix(dir, ".") {
		return dir
	}
	// package names must start with a ./
	return "./" + dir
}

// Expand directories to a map from relative directory path to list of .go files.
func expandDirs(dirs []string) (map[string][]string, error) {
	pwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	out := map[string][]string{}
	for _, dir := range dirs {
		if filepath.IsAbs(dir) {
			dir, _ = filepath.Rel(pwd, dir)
		}
		out[dir], err = filepath.Glob(filepath.Join(dir, "*.go"))
		if err != nil {
			return nil, err
		}
	}
	return out, nil
}
