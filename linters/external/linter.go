package external

import (
	"bytes"

	"github.com/alecthomas/gometalinter/api"
	"github.com/alecthomas/gometalinter/config"
)

type Vars map[string]interface{}

func (v Vars) Copy() Vars {
	out := Vars{}
	for k, v := range v {
		out[k] = v
	}
	return out
}

type Linter struct {
	def *config.ExternalLinterDefinition
	// Configuration variables supported by this external linter.
	//
	// These are interpolated into the command-line and message.
	config Vars
}

// Interpolate vars into template.
func (l *Linter) interpolate(tmpl config.Template, vars Vars) (string, error) {
	w := bytes.NewBuffer(nil)
	err := tmpl.Execute(w, vars)
	if err != nil {
		return "", err
	}
	return w.String(), nil
}

func (l *Linter) Name() string {
	return l.def.Name
}

func (l *Linter) Config() interface{} {
	l.config = map[string]interface{}{}
	return l.config
}

type PackageLinter struct {
	Linter
}

func (p *PackageLinter) LintPackage(ctx api.Context, packages []string) ([]*api.Issue, error) {
	cmdString, err := p.interpolate(p.def.Command.Template, p.config)
	if err != nil {
		return nil, err
	}
	cmd, err := parseCommand(cmdString)
	if err != nil {
		return nil, err
	}
	partitionPathsAsPackages(cmd, packages)
	ExecuteLinter(ctx, &p.Linter, cmd)
}

type FileLinter struct {
	Linter
}

type DirectoryLinter struct {
	Linter
}
