package foo

type StringerVar string

func (myx StringerVar) String() string {
	return string(myx)
}

type Stringer interface {
	String() string
}

type SomeInterface interface {
	FunctionA(StringerVar)
	FunctionB(Stringer) string
}

type SomeVar struct{}

func (i SomeVar) FunctionA(a StringerVar) {
	i.FunctionB(a)
}

func (i SomeVar) FunctionB(a Stringer) string {
	return a.String()
}
