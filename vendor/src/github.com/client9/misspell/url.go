package misspell

import (
	"regexp"
)

// Regexp for URL https://mathiasbynens.be/demo/url-regex
//
// original @imme_emosol (54 chars) has trouble with dashes in hostname
// @(https?|ftp)://(-\.)?([^\s/?\.#-]+\.?)+(/[^\s]*)?$@iS
var urlRE = regexp.MustCompile(`(?i)(https?|ftp)://(-\.)?([^\s/?\.#]+\.?)+(/[^\s]*)?`)

// StripURL attemps to replace URLs with blank spaces, e.g.
//  "xxx http://foo.com/ yyy -> "xxx          yyyy"
func StripURL(s string) string {
	out := []byte(s)
	matches := urlRE.FindAllIndex(out, -1)
	if len(matches) == 0 {
		return s
	}
	for _, idx := range matches {
		for j := idx[0]; j < idx[1]; j++ {
			out[j] = ' '
		}
	}
	return string(out)
}
