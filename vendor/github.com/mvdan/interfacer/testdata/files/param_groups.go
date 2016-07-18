package foo

type Fooer interface {
	Foo()
}

type FooBarer interface {
	Fooer
	Bar()
}

func Separate(fb1 FooBarer, fb2 FooBarer) { // WARN fb1 can be Fooer
	fb1.Foo()
	fb2.Foo()
	fb2.Bar()
}

func Grouped(fb1, fb2 FooBarer) {
	fb1.Foo()
	fb2.Foo()
	fb2.Bar()
}
