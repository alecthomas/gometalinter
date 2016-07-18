package foo

type FooCloser interface {
	Foo()
	Close() error
}

func ShadowArg(fc FooCloser) { // WARN fc can be io.Closer
	fc.Close()
	for {
		fc := 3
		println(fc + 1)
	}
}
