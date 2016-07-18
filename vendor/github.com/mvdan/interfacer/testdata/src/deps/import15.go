// +build !go1.6

package single

type Closer interface {
	Close()
}

type ReadCloser interface {
	Closer
	Read()
}
