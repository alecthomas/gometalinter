package foo

type FooCloser interface {
	Foo()
	Close() error
}

type Barer interface {
	Bar(fc FooCloser) int
}

type st struct{}

func (s st) Bar(fc FooCloser) int {
	return 2
}

func Foo(s st) { // WARN s can be Barer
	_ = s.Bar(nil)
}

func Bar(fc FooCloser) int {
	fc.Close()
	return 3
}
