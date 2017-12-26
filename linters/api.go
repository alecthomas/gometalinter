package linters

import (
	"github.com/alecthomas/gometalinter/api"
)

var Linters = map[string]api.Linter{}

func Register(linter api.Linter) {
	Linters[linter.Name()] = linter
}
