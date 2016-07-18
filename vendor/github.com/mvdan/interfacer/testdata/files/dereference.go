package foo

type Closer interface {
	Close()
}

type st struct{}

func (s *st) Close() {}

func Wrong(s st) { // WARN s can be Closer
	s.Close()
	s = st{}
}

func Dereferenced(s *st) {
	s.Close()
	*s = st{}
}
