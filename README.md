# Go Meta Linter

The number of tools for statically checking Go source for errors and warnings
is impressive.

This is a tool that concurrently runs a whole bunch of those linters and
normalises their output to a standard format. It is intended for use with
editor/IDE integration.

Currently supported linters are: golint, go tool vet, gotype, errcheck,
varcheck and defercheck.

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
```

Run it:

```
$ cd $GOPATH/src/github.com/alecthomas/gometalinter
$ gometalinter
main.go:18:6:warning: exported type Severity should have comment or be unexported
main.go:26:6:warning: exported type Linter should have comment or be unexported
main.go:28:1:warning: exported method Linter.Command should have comment or be unexported
main.go:32:1:warning: exported method Linter.Pattern should have comment or be unexported
main.go:80:6:warning: exported type Issue should have comment or be unexported
main.go:96:6:warning: exported type Issues should have comment or be unexported
```

## Details

```
$ gometalinter --help
usage: gometalinter [<flags>] [<path>]

Aggregate and normalise the output of a whole bunch of Go linters.

Default linters:

  defercheck -> defercheck {paths} -> :PATH:LINE:MESSAGE
  golint -> golint {paths} -> :PATH:LINE:COL:MESSAGE
  vet -> go tool vet {paths} -> :PATH:LINE:MESSAGE
  gotype -> gotype {paths} -> :PATH:LINE:COL:MESSAGE
  errcheck -> errcheck {paths} -> :(?P<path>[^:]+):(?P<line>\d+):(?P<col>\d+)\t(?P<message>.*)
  varcheck -> varcheck {paths} -> :PATH:LINE:MESSAGE

Severity map (default is "error"):

  errcheck -> warning
  golint -> warning
  varcheck -> warning

Flags:
  --help            Show help.
  --install         Attempt to install all known linters.
  --disable-linters=LINTER
                    List of linters to disable.
  --debug           Display messages for failed linters, etc.
  --concurrency=16  Number of concurrent linters to run.
  --linter=NAME:COMMAND:PATTERN
                    Specify a linter.
  --linter-message-overrides=LINTER:MESSAGE
                    Override message from linter. {message} will be expanded to the original message.
  --linter-severity=LINTER:SEVERITY
                    Map of linter severities.

Args:
  [<path>]  Directory to lint.
```

Additional linters can be configured via the command line:

```
$ gometalinter --linter='vet:go tool vet -printfuncs=Infof,Debugf,Warningf,Errorf {paths}:PATH:LINE:MESSAGE' .
stutter.go:12:6:warning: exported type PublicUndocumented should have comment or be unexported
stutter.go:18::error: Repeating defer a.Close() inside function duplicateDefer
stutter.go:9::error: unused global variable unusedGlobal
stutter.go:17:15:warning: error return value not checked (defer a.Close())
stutter.go:18:15:warning: error return value not checked (defer a.Close())
stutter.go:23:6:warning: error return value not checked (doit()           // test for errcheck)
stutter.go:25::error: unreachable code
stutter.go:22::error: missing argument for Printf("%d"): format reads arg 1, have only 0 args
```

