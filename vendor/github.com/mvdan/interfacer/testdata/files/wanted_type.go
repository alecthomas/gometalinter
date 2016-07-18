package foo

type Closer interface {
	Close()
}

type ReadCloser interface {
	Closer
	Read()
}

type Conn struct{}

func (c Conn) Close() {}

func DoClose(c Conn) { // WARN c can be Closer
	c.Close()
}

func DoCloseConn(c Conn) {
	c.Close()
}

func DoCloseConnection(c Conn) {
	c.Close()
}

type bar struct{}

func (f *bar) Close() {}

func barClose(b *bar) {
	b.Close()
}

func DoCloseFoobar(b *bar) { // WARN b can be Closer
	b.Close()
}

type NetConn struct{}

func (n NetConn) Close() {}

func closeConn(conn *NetConn) {
	conn.Close()
}
