// +build go1.8

package main

import "go/doc"

func isPredeclaredIdent(name string) bool {
	return doc.IsPredeclared(name)
}
