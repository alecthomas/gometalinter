// +build go1.6

package single

import "foo/bar"

var _ = bar.Bar

type Closer interface {
	Close()
}

type ReadCloser interface {
	Closer
	Read()
}
