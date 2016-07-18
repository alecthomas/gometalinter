package foo

type Closer interface {
	Close() error
}

type Reader interface {
	Read(p []byte) (n int, err error)
}

type ReadCloser interface {
	Reader
	Closer
}

type st struct{}

func (s st) Read(p []byte) (int, error) {
	return 0, nil
}
func (s st) Close() error {
	return nil
}
func (s st) Other() {}

func FooCloser(c Closer) {
	c.Close()
}

func FooSt(s st) {
	s.Other()
}

func Bar(s st) {
	s.Close()
	FooSt(s)
}

func BarWrong(s st) { // WARN s can be io.Closer
	s.Close()
	FooCloser(s)
}

func extra(n int, cs ...Closer) {}

func ArgExtraWrong(s1 st) { // WARN s1 can be io.Closer
	var s2 st
	s1.Close()
	s2.Close()
	extra(3, s1, s2)
}

func Assigned(s st) {
	s.Close()
	var s2 st
	s2 = s
	_ = s2
}

func Declared(s st) {
	s.Close()
	var s2 st = s
	_ = s2
}

func AssignedIface(s st) { // WARN s can be io.Closer
	s.Close()
	var c Closer
	c = s
	_ = c
}

func AssignedIfaceDiff(s st) { // WARN s can be io.ReadCloser
	s.Close()
	var r Reader
	r = s
	_ = r
}

func doRead(r Reader) {
	b := make([]byte, 10)
	r.Read(b)
}

func ArgIfaceDiff(s st) { // WARN s can be io.ReadCloser
	s.Close()
	doRead(s)
}
