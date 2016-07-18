package foo

type Closer interface {
	Close() error
}

type ReadCloser interface {
	Closer
	Read() (int, error)
}

func Results(rc ReadCloser) {
	_, _ = rc.Read()
	err := rc.Close()
	println(err)
}

func ResultsWrong(rc ReadCloser) { // WARN rc can be io.Closer
	err := rc.Close()
	println(err)
}

type argBad struct{}

func (a argBad) Read() (string, error) {
	return "", nil
}

func (a argBad) Write() error {
	return nil
}

func (a argBad) Close() int {
	return 0
}

func ResultsMismatchNumber(a argBad) {
	_ = a.Write()
}

func ResultsMismatchType(a argBad) {
	s, _ := a.Read()
	println(s)
}

func ResultsMismatchTypes(a, b argBad) {
	r1, r2 := a.Close(), b.Close()
	println(r1, r2)
}
