package foo

type Closer interface {
	Close()
}

type ReadCloser interface {
	Read(p []byte) (n int, err error)
	Closer
}

func Wrong(rc ReadCloser) { // WARN rc can be Closer
	rc.Close()
}

func receiver(f func(ReadCloser)) {
	var rc ReadCloser
	f(rc)
}

func Correct(rc ReadCloser) {
	rc.Close()
}

func CorrectUse() {
	receiver(Correct)
}

var holder func(ReadCloser)

func Correct2(rc ReadCloser) {
	rc.Close()
}

func CorrectAssign() {
	holder = Correct2
}

func CorrectLit() {
	f := func(rc ReadCloser) {
		rc.Close()
	}
	receiver(f)
}

func CorrectLitAssign() {
	f := func(rc ReadCloser) {
		rc.Close()
	}
	var f2 func(ReadCloser)
	f2 = f
	f2(nil)
}

func CorrectLitDecl() {
	f := func(rc ReadCloser) {
		rc.Close()
	}
	var f2 func(ReadCloser) = f
	f2(nil)
}
