package foo

type Closer interface {
	Close()
}

func DoCloseOther(rc interface { // WARN rc can be Closer
	Close()
	Read()
}) {
	rc.Close()
}
