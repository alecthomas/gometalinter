package foo

type mint int

func (m mint) String() string {
	return ""
}

func Unary(m mint) {
	m.String()
	_ = -m
}

func BinaryLeft(m mint) {
	m.String()
	_ = m + 3
}

func BinaryRight(m mint) {
	m.String()
	_ = 3 + m
}

func IncDec(m mint) {
	m.String()
	m++
}

type marr [3]int

func (m marr) String() string {
	return ""
}

func Index(m marr) {
	m.String()
	_ = m[1]
}

func BinaryNonIface(m mint) {
	if m > 3 {
		m.String()
	}
}
