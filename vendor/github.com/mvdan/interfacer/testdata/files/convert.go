package foo

type mint int

func (m mint) Close() error {
	return nil
}

type mint2 mint

func ConvertStruct(m mint) {
	m.Close()
	_ = mint2(m)
}

func ConvertBasic(m mint) {
	m.Close()
	println(int(m))
}
