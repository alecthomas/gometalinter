package external

import (
	"github.com/alecthomas/gometalinter/config"
)

type Linter struct {
	def    *config.ExternalLinterDefinition
	config map[string]string
}

func (l *Linter) Name() string {
	return l.def.Name
}

func (l *Linter) Config() interface{} {
	l.config = map[string]string{}
	return l.config
}

type PackageLinter struct {
	Linter
}

type FileLinter struct {
	Linter
}

type DirectoryLinter struct {
	Linter
}
