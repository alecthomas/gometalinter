package foo

type Namer interface {
	Name() string
}

type st struct{}

func (s st) Name() string {
	return ""
}

type MyPathFunc func(path string, s st) error
type MyPathFunc2 func(path string, s st) error

func Impl(path string, s st) error {
	s.Name()
	return nil
}

type MyIface interface {
	FooBar(s *st)
}
type MyIface2 interface {
	MyIface
}

type impl struct{}

func (im impl) FooBar(s *st) {}

func FooWrong(im impl) { // WARN im can be MyIface
	im.FooBar(nil)
}

type FooBarFunc func(s st)
