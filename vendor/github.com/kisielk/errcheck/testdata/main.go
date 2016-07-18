package main

import "fmt"

func a() error {
	fmt.Println("this function returns an error") // UNCHECKED
	return nil
}

func b() (int, error) {
	fmt.Println("this function returns an int and an error") // UNCHECKED
	return 0, nil
}

func c() int {
	fmt.Println("this function returns an int") // UNCHECKED
	return 7
}

func rec() {
	defer func() {
		recover()     // UNCHECKED
		_ = recover() // BLANK
	}()
	defer recover() // UNCHECKED
}

type MyError string

func (e MyError) Error() string {
	return string(e)
}

func customError() error {
	return MyError("an error occurred")
}

func customConcreteError() MyError {
	return MyError("an error occurred")
}

func customConcreteErrorTuple() (int, MyError) {
	return 0, MyError("an error occurred")
}

type MyPointerError string

func (e *MyPointerError) Error() string {
	return string(*e)
}

func customPointerError() *MyPointerError {
	e := MyPointerError("an error occurred")
	return &e
}

func customPointerErrorTuple() (int, *MyPointerError) {
	e := MyPointerError("an error occurred")
	return 0, &e
}

func main() {
	// Single error return
	_ = a() // BLANK
	a()     // UNCHECKED

	// Return another value and an error
	_, _ = b() // BLANK
	b()        // UNCHECKED

	// Return a custom error type
	_ = customError() // BLANK
	customError()     // UNCHECKED

	// Return a custom concrete error type
	_ = customConcreteError()         // BLANK
	customConcreteError()             // UNCHECKED
	_, _ = customConcreteErrorTuple() // BLANK
	customConcreteErrorTuple()        // UNCHECKED

	// Return a custom pointer error type
	_ = customPointerError()         // BLANK
	customPointerError()             // UNCHECKED
	_, _ = customPointerErrorTuple() // BLANK
	customPointerErrorTuple()        // UNCHECKED

	// Method with a single error return
	x := t{}
	_ = x.a() // BLANK
	x.a()     // UNCHECKED

	// Method call on a struct member
	y := u{x}
	_ = y.t.a() // BLANK
	y.t.a()     // UNCHECKED

	m1 := map[string]func() error{"a": a}
	_ = m1["a"]() // BLANK
	m1["a"]()     // UNCHECKED

	// Additional cases for assigning errors to blank identifier
	z, _ := b()    // BLANK
	_, w := a(), 5 // BLANK

	// Assign non error to blank identifier
	_ = c()

	_ = z + w // Avoid complaints about unused variables

	// Type assertions
	var i interface{}
	s1 := i.(string)    // ASSERT
	s1 = i.(string)     // ASSERT
	s2, _ := i.(string) // ASSERT
	s2, _ = i.(string)  // ASSERT
	s3, ok := i.(string)
	s3, ok = i.(string)
	switch s4 := i.(type) {
	case string:
		_ = s4
	}
	_, _, _, _ = s1, s2, s3, ok

	// Goroutine
	go a()    // UNCHECKED
	defer a() // UNCHECKED
}
