package pkg2

type Closer interface {
	Close()
}

type ReadCloser interface {
	Closer
	Read()
}

func BasicWrong(rc ReadCloser) { // WARN rc can be Closer
	rc.Close()
}
