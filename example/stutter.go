package main

import (
	"fmt"
	"io"
)

var (
	unusedGlobal = true
)

type MyStruct struct {
	Unused bool
}

type PublicUndocumented int // test for golint

func doit() error { return nil }

func duplicateDefer(a io.Closer) {
	defer a.Close()
	defer a.Close()
}

func main() {
	fmt.Printf("%d") // test for "go vet"
	doit()           // test for errcheck
	return
	println("lalal")
}
