# Go Meta Linter

The number of tools for statically checking Go source for errors and warnings
is impressive.

This is a tool that concurrently runs a whole bunch of those linters and
normalises their output to a standard format. It is intended for use with
editor/IDE integration.

Currently supported linters are: golint, go tool vet, gotype, errcheck,
varcheck and defercheck. Additional linters can be added through the
command line with `--linter=NAME:COMMAND:PATTERN` (see [below](#details)).

## Quickstart

Install gometalinter:

```
$ go get github.com/alecthomas/gometalinter
```

Install all known linters:

```
$ gometalinter --install
Installing golint -> go get github.com/golang/lint/golint
Installing gotype -> go get code.google.com/p/go.tools/cmd/gotype
Installing errcheck -> go get github.com/kisielk/errcheck
Installing defercheck -> go get github.com/opennota/check/cmd/defercheck
Installing varcheck -> go get github.com/opennota/check/cmd/varcheck
Installing structcheck -> go get github.com/opennota/check/cmd/structcheck
```

Run it:

```
$ cd $GOPATH/src/github.com/alecthomas/gometalinter/example
$ gometalinter
stutter.go:22::error: Repeating defer a.Close() inside function duplicateDefer
stutter.go:12:6:warning: exported type MyStruct should have comment or be unexported
stutter.go:16:6:warning: exported type PublicUndocumented should have comment or be unexported
stutter.go:21:15:warning: error return value not checked (defer a.Close())
stutter.go:22:15:warning: error return value not checked (defer a.Close())
stutter.go:27:6:warning: error return value not checked (doit()           // test for errcheck)
stutter.go:9::warning: unused global variable unusedGlobal
stutter.go:13::warning: unused struct field MyStruct.Unused
stutter.go:29::error: unreachable code
stutter.go:26::error: missing argument for Printf("%d"): format reads arg 1, have only 0 args
```

## Details

```
$ gometalinter --help
usage: gometalinter [<flags>] [<path>]

Aggregate and normalise the output of a whole bunch of Go linters.

Default linters:

  varcheck -> varcheck {path} -> :PATH:LINE:MESSAGE
  structcheck -> structcheck {path} -> :PATH:LINE:MESSAGE
  defercheck -> defercheck {path} -> :PATH:LINE:MESSAGE
  golint -> golint {path} -> :PATH:LINE:COL:MESSAGE
  vet -> go tool vet {path} -> :PATH:LINE:MESSAGE
  gotype -> gotype {path} -> :PATH:LINE:COL:MESSAGE
  errcheck -> errcheck {path} -> :(?P<path>[^:]+):(?P<line>\d+):(?P<col>\d+)\t(?P<message>.*)

Severity override map (default is "error"):

  errcheck -> warning
  golint -> warning
  varcheck -> warning
  structcheck -> warning

Flags:
  --help       Show help.
  --install    Attempt to install all known linters.
  -D, --disable=LINTER  List of linters to disable.
  -d, --debug  Display messages for failed linters, etc.
  -j, --concurrency=16  Number of concurrent linters to run.
  --linter=NAME:COMMAND:PATTERN
               Specify a linter.
  --message-overrides=LINTER:MESSAGE
               Override message from linter. {message} will be expanded to the original message.
  --severity=LINTER:SEVERITY
               Map of linter severities.

Args:
  [<path>]  Directory to lint.
```

Additional linters can be configured via the command line:

```
$ gometalinter --linter='vet:go tool vet -printfuncs=Infof,Debugf,Warningf,Errorf {paths}:PATH:LINE:MESSAGE' .
stutter.go:22::error: Repeating defer a.Close() inside function duplicateDefer
stutter.go:21:15:warning: error return value not checked (defer a.Close())
stutter.go:22:15:warning: error return value not checked (defer a.Close())
stutter.go:27:6:warning: error return value not checked (doit()           // test for errcheck)
stutter.go:9::warning: unused global variable unusedGlobal
stutter.go:13::warning: unused struct field MyStruct.Unused
stutter.go:12:6:warning: exported type MyStruct should have comment or be unexported
stutter.go:16:6:warning: exported type PublicUndocumented should have comment or be unexported
```

