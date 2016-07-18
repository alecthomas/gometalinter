# interfacer

[![Build Status](https://travis-ci.org/mvdan/interfacer.svg?branch=master)](https://travis-ci.org/mvdan/interfacer)

A linter that suggests interface types. In other words, it warns about
the usage of types that are more specific than necessary.

	go get -u github.com/mvdan/interfacer/cmd/interfacer

### Usage

```go
func ProcessInput(f *os.File) error {
        b, err := ioutil.ReadAll(f)
        if err != nil {
                return err
        }
        return processBytes(b)
}
```

```sh
$ interfacer ./...
foo.go:10:19: f can be io.Reader
```

### Basic idea

This package relies on `go/types` for the heavy lifting: name
resolution, constant folding and type inference. It also uses
`go/loader` to resolve the packages specified by import paths.

It inspects the parameters of your functions to see if they fit an
interface type that is less specific than the current type.

The example above illustrates this point. Overly specific interfaces
also trigger a warning - if `f` were an `io.ReadCloser`, the same
message would appear.

It suggests interface types defined both in `std` and in your packages.

### False positives

To avoid false positives, it never does any suggestions on functions
that may be implementing an interface method or a named function type.

It also skips parameters passed by value (excluding pointers and
interfaces) on unexported functions, since that would introduce extra
allocations where they are usually not worth the tradeoff.

### Suppressing warnings

If a suggestion is technically correct but doesn't make sense, you can
still suppress the warning by mentioning the type in the function name:

```go
func ProcessInputFile(f *os.File) error {
	// use as an io.Reader
}
```
