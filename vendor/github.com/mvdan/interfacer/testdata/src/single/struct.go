package single

type st struct{}

func (s *st) Close() {}

func (s *st) Basic(c Closer) {
	c.Close()
}

func (s *st) BasicWrong(rc ReadCloser) { // WARN rc can be Closer
	rc.Close()
}
