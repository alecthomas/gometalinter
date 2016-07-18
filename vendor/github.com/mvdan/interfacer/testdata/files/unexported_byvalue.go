package foo

type st struct{}

func (s st) String() string {
	return "foo"
}

func Exported(s st) string { // WARN s can be fmt.Stringer
	return s.String()
}

func unexported(s st) string {
	return s.String()
}
