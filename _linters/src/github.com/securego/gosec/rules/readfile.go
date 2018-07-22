// (c) Copyright 2016 Hewlett Packard Enterprise Development LP
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package rules

import (
	"go/ast"
	"go/types"

	"github.com/securego/gosec"
)

type readfile struct {
	gosec.MetaData
	gosec.CallList
}

// ID returns the identifier for this rule
func (r *readfile) ID() string {
	return r.MetaData.ID
}

// Match inspects AST nodes to determine if the match the methods `os.Open` or `ioutil.ReadFile`
func (r *readfile) Match(n ast.Node, c *gosec.Context) (*gosec.Issue, error) {
	if node := r.ContainsCallExpr(n, c); node != nil {
		for _, arg := range node.Args {
			if ident, ok := arg.(*ast.Ident); ok {
				obj := c.Info.ObjectOf(ident)
				if _, ok := obj.(*types.Var); ok && !gosec.TryResolve(ident, c) {
					return gosec.NewIssue(c, n, r.ID(), r.What, r.Severity, r.Confidence), nil
				}
			}
		}
	}
	return nil, nil
}

// NewReadFile detects cases where we read files
func NewReadFile(id string, conf gosec.Config) (gosec.Rule, []ast.Node) {
	rule := &readfile{
		CallList: gosec.NewCallList(),
		MetaData: gosec.MetaData{
			ID:         id,
			What:       "Potential file inclusion via variable",
			Severity:   gosec.Medium,
			Confidence: gosec.High,
		},
	}
	rule.Add("io/ioutil", "ReadFile")
	rule.Add("os", "Open")
	return rule, []ast.Node{(*ast.CallExpr)(nil)}
}
