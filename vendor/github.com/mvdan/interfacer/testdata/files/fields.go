package foo

type st struct {
	field int
	m     map[int]int
}

func (s *st) Close() error {
	return nil
}

func (s *st) Lock() {}

func (s *st) Unlock() {}

func Foo(s *st) {
	s.Close()
	s.field = 3
}

func FooWrong(s *st) { // WARN s can be io.Closer
	s.Close()
}

type st2 struct {
	st1 *st
}

func (s *st2) Close() error {
	return nil
}

func Foo2(s *st2) {
	s.Close()
	s.st1.field = 3
}

func Foo2Wrong(s *st2) { // WARN s can be io.Closer
	s.Close()
}

func FooPassed(s *st) {
	s.Close()
	s2 := s
	s2.field = 2
}

func FooPassedWrong(s *st) { // WARN s can be io.Closer
	s.Close()
	s2 := s
	s2.Close()
}

func FooBuiltinArg(s *st) func(int) {
	s.Lock()
	s.Unlock()
	delete(s.m, 3)
	return nil
}
