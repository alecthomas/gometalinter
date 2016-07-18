package def

type ReadCloser interface {
	Read([]byte) (int, error)
	Close() error
}

type FooFunc func(ReadCloser, int) int

var SomeVar int = 3

func SomeFunc() {}
