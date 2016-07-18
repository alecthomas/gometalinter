package foo

type Closer interface {
	Close()
}

type ReadCloser interface {
	Closer
	Read()
}

type myFunc func(rc ReadCloser, err error) int

func MyFuncImpl(rc ReadCloser, err error) int {
	rc.Close()
	return 0
}

func MyFuncWrong(rc ReadCloser, err error) { // WARN rc can be Closer
	rc.Close()
}

type myIface interface {
	Foo(rc ReadCloser, i int64)
}

func FooImpl(rc ReadCloser, i int64) {
	rc.Close()
}

type st struct{}

func (s *st) Foo(rc ReadCloser, i int64) {}

func DoNotSuggestUnexportedIface(s *st, rc ReadCloser) {
	a := int64(3)
	s.Foo(rc, a)
}
