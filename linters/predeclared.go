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
	Register(&PredeclaredLinter{})
}

type PredeclaredConfig struct {
	// Include method names and field names while checking
	Qualified bool
	// List of predeclared identifiers to not report on.
	IgnoredIdents []string
}

type PredeclaredLinter struct {
	config predeclared.Config
}

func (p *PredeclaredLinter) Name() string { return "predeclared" }
func (p *PredeclaredLinter) Config(unmarshal api.ConfigUnmarshaller) error {
	config := PredeclaredConfig{}
	err := unmarshal(&config)
	if err != nil {
		return err
	}
	p.config = predeclared.Config{
		Qualified:     config.Qualified,
		IgnoredIdents: map[string]bool{},
	}
	for _, ident := range config.IgnoredIdents {
		p.config.IgnoredIdents[ident] = true
	}
	return nil
}
func (p *PredeclaredLinter) LintAST(fset *token.FileSet, files []*ast.File) ([]*api.Issue, error) {
	issues := []*api.Issue{}
	for _, file := range files {
		pissues := predeclared.ProcessFile(&p.config, fset, file)
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
	}
	return issues, nil
}
