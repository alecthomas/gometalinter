package foo

type EmptyIface interface{}

type UninterestingMethods interface {
	Foo() error
	bar() int
}

type InterestingUnexported interface {
	Foo(f string) error
	bar(f string) int
}

type st struct{}

func (s st) Foo(f string) {}

func (s st) nonExported() {}

func Bar(s st) {
	s.Foo("")
}

type NonInterestingFunc func() error
