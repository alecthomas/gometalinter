package foo

type Closer interface {
	Close()
}

type ReadCloser interface {
	Closer
	Read()
}

func Transitive(rc ReadCloser) { // WARN rc can be Closer
	a := rc
	b := a
	c := b
	c.Close()
}
