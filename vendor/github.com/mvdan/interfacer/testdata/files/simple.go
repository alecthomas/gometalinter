package foo

func Empty() {
}

type Closer interface {
	Close()
}

type ReadCloser interface {
	Closer
	Read()
}

func Basic(c Closer) {
	c.Close()
}

func BasicInteresting(rc ReadCloser) {
	rc.Read()
	rc.Close()
}

func BasicWrong(rc ReadCloser) { // WARN rc can be Closer
	rc.Close()
}

type st struct{}

func (s *st) Basic(c Closer) {
	c.Close()
}

func (s *st) BasicWrong(rc ReadCloser) { // WARN rc can be Closer
	rc.Close()
}
