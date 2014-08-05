# Go Meta Linter

The number of tools for statically checking Go source for errors and warnings
is impressive.

This is a tool that concurrently runs a whole bunch of those linters and
normalises their output to a standard format. It is intended for use with
editor/IDE integration.

Currently supported linters are: golint, go tool vet, gotype, errcheck,
varcheck and defercheck: 

```
$ gometalinter --help
usage: gometalinter [<flags>] <paths>

Aggregate and normalise the output of a whole bunch of Go linters.

Default linters:

  golint -> golint {paths} -> PATH:LINE:COL:MESSAGE
  vet -> go tool vet {paths} -> PATH:LINE:MESSAGE
  gotype -> gotype {paths} -> PATH:LINE:COL:MESSAGE
  errcheck -> errcheck {paths} -> (?P<path>[^:]+):(?P<line>\d+):(?P<col>\d+)\t(?P<message>.*)
  varcheck -> varcheck {paths} -> PATH:LINE:MESSAGE
  defercheck -> defercheck {paths} -> PATH:LINE:MESSAGE

Flags:
  --help            Show help.
  --disable-linters=LINTER
                    List of linters to disable.
  --debug           Display messages for failed linters, etc.
  --concurrency=16  Number of concurrent linters to run.
  --linter=NAME:COMMAND:PATTERN
                    Specify a linter.
  --linter-message-overrides=LINTER:MESSAGE
                    Override message from linter. {message} will be expanded to
                    the original message.
  --linter-severity=LINTER:SEVERITY
                    Map of linter severities.

Args:
  <paths>  Directories to lint.
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

