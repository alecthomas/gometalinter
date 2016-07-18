package foo

type Closer interface {
	Close()
}

type ReadCloser interface {
	Closer
	Read()
}

func Recursive(rc ReadCloser) {
	rc.Read()
	rc.Close()
	Recursive(rc)
}

func RecursiveWrong(rc ReadCloser) { // WARN rc can be Closer
	rc.Close()
	RecursiveWrong(rc)
}
