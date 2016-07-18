package foo

type Closer interface {
	Close() error
}

type Reader interface {
	Read(p []byte) (n int, err error)
}

type ReadCloser interface {
	Reader
	Closer
}

type Seeker interface {
	Seek(int64, int) (int64, error)
}

type ReadSeeker interface {
	Reader
	Seeker
}

func Args(rc ReadCloser) {
	b := make([]byte, 10)
	rc.Read(b)
	rc.Close()
}

func ArgsWrong(rc ReadCloser) { // WARN rc can be io.Reader
	b := make([]byte, 10)
	rc.Read(b)
}

func ArgsLit(rs ReadSeeker) {
	b := make([]byte, 10)
	rs.Read(b)
	rs.Seek(20, 0)
}

func ArgsLitWrong(rs ReadSeeker) { // WARN rs can be io.Seeker
	rs.Seek(20, 0)
}

func ArgsLit2(rs ReadSeeker) {
	rs.Read([]byte{})
	rs.Seek(20, 0)
}

func ArgsLit2Wrong(rs ReadSeeker) { // WARN rs can be io.Reader
	rs.Read([]byte{})
}

func ArgsNil(rs ReadSeeker) {
	rs.Read(nil)
	rs.Seek(20, 0)
}

func ArgsNilWrong(rs ReadSeeker) { // WARN rs can be io.Reader
	rs.Read(nil)
}

type st struct{}

func (s st) Args(rc ReadCloser) {
	var b []byte
	rc.Read(b)
	rc.Close()
}

func (s st) ArgsWrong(rc ReadCloser) { // WARN rc can be io.Reader
	b := make([]byte, 10)
	rc.Read(b)
}

type argBad struct{}

func (a argBad) Read(n int) (int, error) {
	return 0, nil
}

func (a argBad) Close(n int) error {
	return nil
}

type argGood struct{}

func (a argGood) Read(p []byte) (int, error) {
	return 0, nil
}

func ArgsMismatch(a argBad) {
	a.Read(10)
}

func ArgsMatch(a argGood) { // WARN a can be io.Reader
	b := make([]byte, 10)
	a.Read(b)
}

func ArgsMismatchNum(a argBad) {
	a.Close(3)
}

func ArgsExtra() {
	println(12, "foo")
}

func BuiltinExtra(s string) {
	i := 2
	b := make([]byte, i)
	_ = append(b, s...)
}
