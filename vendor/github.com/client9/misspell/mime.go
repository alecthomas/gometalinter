package misspell

import (
	"path/filepath"
	"strings"
)

// The number of possible binary formats is very large
// items that might be checked into a repo or be an
// artifact of a build.  Additions welcome.
//
// Golang's internal table is very small and can't be
// relied on.  Even then things like ".js" have a mime
// type of "application/javascipt" which isn't very helpful.
//
var binary = map[string]bool{
	".a":     true, // archive
	".bin":   true, // binary
	".bz2":   true, // compression
	".class": true, // Java class file
	".dll":   true, // shared library
	".exe":   true, // binary
	".gif":   true, // image
	".gz":    true, // compression
	".ico":   true, // image
	".jar":   true, // archive
	".jpeg":  true, // image
	".jpg":   true, // image
	".mp3":   true, // audio
	".mp4":   true, // video
	".mpeg":  true, // video
	".o":     true, // object file
	".pdf":   true, // pdf -- might be possible to use this later
	".png":   true, // image
	".pyc":   true, // Python bytecode
	".pyo":   true, // Python bytecode
	".so":    true, // shared library
	".swp":   true, // vim swap file
	".tar":   true, // archive
	".tiff":  true, // image
	".woff":  true, // font
	".woff2": true, // font
	".xz":    true, // compression
	".z":     true, // compression
	".zip":   true, // archive
}

// IsBinaryFile returns true if the file is likely to be binary
//
// Better heuristics could be done here, in particular a binary
// file is unlikely to be UTF-8 encoded.  However this is cheap
// and will solve the immediate need of making sure common
// binary formats are not corrupted by mistake.
func IsBinaryFile(s string) bool {
	return binary[strings.ToLower(filepath.Ext(s))]
}

var scm = map[string]bool{
	".bzr": true,
	".git": true,
	".hg":  true,
	".svn": true,
	"CVS":  true,
}

// IsSCMPath returns true if the path is likely part of a (private) SCM
//  directory.  E.g.  ./git/something  = true
func IsSCMPath(s string) bool {
	parts := strings.Split(s, string(filepath.Separator))
	for _, dir := range parts {
		if scm[dir] {
			return true
		}
	}
	return false
}
