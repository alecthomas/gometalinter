package linters

import (
	"fmt"
	"go/ast"
	"go/token"

	predeclared "github.com/nishanths/predeclared/api"

	"github.com/alecthomas/gometalinter/api"
)

var _ api.ASTLinter = &PredeclaredLinter{}

func init() {
	Register(func() api.Linter { return &PredeclaredLinter{} })
}

type PredeclaredConfig struct {
	// Include method names and field names while checking
	Qualified bool `toml:"qualified"`
	// List of predeclared identifiers to not report on.
	IgnoredIdents []string `toml:"ignored_idents"`
}

type PredeclaredLinter struct {
	config PredeclaredConfig
}

func (p *PredeclaredLinter) Name() string        { return "predeclared" }
func (p *PredeclaredLinter) Config() interface{} { return &p.config }
func (p *PredeclaredLinter) LintAST(ctx api.Context, fset *token.FileSet, files map[string]*ast.File) ([]*api.Issue, error) {
	config := p.makeConfig()
	pissues := []predeclared.Issue{}
	// Process files.
	for _, file := range files {
		pissues = append(pissues, predeclared.ProcessFile(config, fset, file)...)
	}
	// Convert predeclared issues to issues.
	issues := []*api.Issue{}
	for _, pissue := range pissues {
		pos := pissue.Pos()
		issue := &api.Issue{
			Linter:   p.Name(),
			Col:      pos.Column,
			Line:     pos.Line,
			Path:     pos.Filename,
			Message:  fmt.Sprintf("%s %q has same name as predeclared identifier", pissue.Kind, pissue.Ident.Name),
			Severity: api.Warning,
		}
		issues = append(issues, issue)
	}
	return issues, nil
}

func (p *PredeclaredLinter) makeConfig() *predeclared.Config {
	config := &predeclared.Config{
		Qualified:     p.config.Qualified,
		IgnoredIdents: map[string]bool{},
	}
	for _, id := range p.config.IgnoredIdents {
		config.IgnoredIdents[id] = true
	}
	return config
}
