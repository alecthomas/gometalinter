package foo

type Reader interface {
	Read(p []byte) (n int, err error)
}

type Seeker interface {
	Seek(int64, int) (int64, error)
}

type ReadSeeker interface {
	Reader
	Seeker
}

const offset = 1

func Const(s Seeker) {
	var whence int = 0
	s.Seek(offset, whence)
}

func ConstWrong(rs ReadSeeker) { // WARN rs can be io.Seeker
	var whence int = 0
	rs.Seek(offset, whence)
}

func LocalConst(s Seeker) {
	const offset2 = 2
	var whence int = 0
	s.Seek(offset2, whence)
}

func LocalConstWrong(rs ReadSeeker) { // WARN rs can be io.Seeker
	const offset2 = 2
	var whence int = 0
	rs.Seek(offset2, whence)
}

func AssignFromConst() {
	var i int
	i = offset
	println(i)
}
