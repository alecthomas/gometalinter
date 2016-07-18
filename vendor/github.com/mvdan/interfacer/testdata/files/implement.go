package foo

type Closer interface {
	Close()
}

type ReadCloser interface {
	Closer
	Read()
}

type MyFunc func(rc ReadCloser, err error) bool

func MyFuncImpl(rc ReadCloser, err error) bool {
	rc.Close()
	return false
}

func MyFuncWrong(rc ReadCloser, err error) { // WARN rc can be Closer
	rc.Close()
}
