package foo

type MyCloser interface {
	Close() error
}

func Foo(c MyCloser) {
	c.Close()
}
