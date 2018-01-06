package linters

import (
	"github.com/alecthomas/gometalinter/api"
)

var Linters = map[string]api.LinterFactory{}

func Register(linter api.LinterFactory) {
	Linters[linter().Name()] = linter
}
