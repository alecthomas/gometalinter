// +build go1.8

package api

import "go/doc"

func isPredeclaredIdent(name string) bool {
	return doc.IsPredeclared(name)
}
