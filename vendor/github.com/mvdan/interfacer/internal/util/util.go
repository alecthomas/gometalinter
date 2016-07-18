// Copyright (c) 2015, Daniel Martí <mvdan@mvdan.cc>
// See LICENSE for licensing information

package util

import (
	"unicode"
	"unicode/utf8"
)

type ByAlph []string

func (l ByAlph) Len() int           { return len(l) }
func (l ByAlph) Less(i, j int) bool { return l[i] < l[j] }
func (l ByAlph) Swap(i, j int)      { l[i], l[j] = l[j], l[i] }

func Exported(name string) bool {
	ch, _ := utf8.DecodeRuneInString(name)
	return unicode.IsUpper(ch)
}
