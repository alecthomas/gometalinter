package skip

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

func BasicWrong(rc ReadCloser) { // WARN rc can be Closer
	rc.Close()
}
