package foo

type Closer interface {
	Close()
}

type ReadCloser interface {
	Closer
	Read()
}

func Basic(s string) {
	_ = s
}

func BasicWrong(rc ReadCloser) { // WARN rc can be Closer
	rc.Close()
}

func Array(ints [3]int) {}

func ArrayIface(rcs [3]ReadCloser) {
	rcs[1].Close()
}

func Slice(ints []int) {}

func SliceIface(rcs []ReadCloser) {
	rcs[1].Close()
}

func TypeConversion(i int) int64 {
	return int64(i)
}

func LocalType() {
	type str string
}
