# Go Meta Linter

The number of tools for statically checking Go source for errors and warnings
is impressive.

This is a tool that concurrently runs a whole bunch of those linters and
normalises their output to a standard format. It is intended for use with
editor/IDE integration.

Currently supported linters are listed below. Additional linters can be added through the
command line with `--linter=NAME:COMMAND:PATTERN` (see [below](#details)).

## Editor integration

- [SublimeLinter plugin](https://github.com/alecthomas/SublimeLinter-contrib-gometalinter).

## Quickstart

Install gometalinter:

```
$ go get github.com/alecthomas/gometalinter
```

Install all known linters:

```
$ gometalinter --install
Installing errcheck -> go get github.com/kisielk/errcheck
Installing structcheck -> go get github.com/opennota/check/cmd/structcheck
Installing deadcode -> go get github.com/remyoudompheng/go-misc/deadcode
Installing golint -> go get github.com/golang/lint/golint
Installing gotype -> go get golang.org/x/tools/cmd/gotype
Installing defercheck -> go get github.com/opennota/check/cmd/defercheck
Installing varcheck -> go get github.com/opennota/check/cmd/varcheck
Installing gocyclo -> go get github.com/fzipp/gocyclo
Installing go-nyet -> go get github.com/barakmich/go-nyet
```

Run it:

```
$ cd $GOPATH/src/github.com/alecthomas/gometalinter/example
$ gometalinter ./...
stutter.go:13::warning: unused struct field MyStruct.Unused (structcheck)
stutter.go:9::warning: unused global variable unusedGlobal (varcheck)
stutter.go:12:6:warning: exported type MyStruct should have comment or be unexported (golint)
stutter.go:16:6:warning: exported type PublicUndocumented should have comment or be unexported (golint)
stutter.go:22::error: Repeating defer a.Close() inside function duplicateDefer (defercheck)
stutter.go:8:1:warning: unusedGlobal is unused (deadcode)
stutter.go:12:1:warning: MyStruct is unused (deadcode)
stutter.go:16:1:warning: PublicUndocumented is unused (deadcode)
stutter.go:20:1:warning: duplicateDefer is unused (deadcode)
stutter.go:21:15:warning: error return value not checked (defer a.Close()) (errcheck)
stutter.go:22:15:warning: error return value not checked (defer a.Close()) (errcheck)
stutter.go:27:6:warning: error return value not checked (doit()           // test for errcheck) (errcheck)
stutter.go:29::error: unreachable code (vet)
stutter.go:26::error: missing argument for Printf("%d"): format reads arg 1, have only 0 args (vet)
```


Gometalinter also supports the commonly seen `<path>/...` recursive path
format. Note that this can be *very* slow.

## Troubleshooting

Sometimes gometalinter will not report issues that you think it should. There
are two things to try in that case:

### 1. Update to the latest build of gometalinter and all linters

    go get -u github.com/alecthomas/gometalinter
    gometalinter --install --update

If you're lucky, this will fix the problem.

### 2. Analyse the debug output

If that doesn't help, the problem may be elsewhere (in no particular order):

1. Upstream linter has changed its output or semantics.
2. gometalinter is not invoking the tool correctly.
3. gometalinter regular expression matches are not correct for a linter.
4. Linter is exceeding the deadline.

To find out what's going on run in debug mode:

    gometalinter --debug

This will show all output from the linters and should indicate why it is
failing. If the problem looks like a bug, please file an issue and paste the
debug output.

## Details

```
$ gometalinter --help
usage: gometalinter [<flags>] [<path>]

Aggregate and normalise the output of a whole bunch of Go linters.

Default linters:

  gotype (golang.org/x/tools/cmd/gotype)
      gotype {tests=-a} {path}
      :PATH:LINE:COL:MESSAGE
  varcheck (github.com/opennota/check/cmd/varcheck)
      varcheck {path}
      :^(?:[^:]+: )?(?P<path>[^:]+):(?P<line>\d+):(?P<col>\d+):\s*(?P<message>\w+)$
  deadcode (github.com/remyoudompheng/go-misc/deadcode)
      deadcode {path}
      :deadcode: (?P<path>[^:]+):(?P<line>\d+):(?P<col>\d+):\s*(?P<message>.*)
  golint (github.com/golang/lint/golint)
      golint {path}
      :PATH:LINE:COL:MESSAGE
  errcheck (github.com/alecthomas/errcheck)
      errcheck {path}
      :^(?P<path>[^:]+):(?P<line>\d+):(?P<col>\d+)\t(?P<message>.*)$
  structcheck (github.com/opennota/check/cmd/structcheck)
      structcheck {tests=-t} {path}
      :^(?:[^:]+: )?(?P<path>[^:]+):(?P<line>\d+):\s*(?P<message>[\w.]+)$
  defercheck (github.com/opennota/check/cmd/defercheck)
      defercheck {path}
      :PATH:LINE:MESSAGE
  gocyclo (github.com/alecthomas/gocyclo)
      gocyclo -over {mincyclo} {path}
      :^(?P<cyclo>\d+)\s+\S+\s(?P<function>\S+)\s+(?P<path>[^:]+):(?P<line>\d+):(?P<col>\d+)
  go-nyet (github.com/barakmich/go-nyet)
      go-nyet {path}
      :PATH:LINE:COL:MESSAGE
  vet ()
      go vet {path}
      :PATH:LINE:MESSAGE

Severity override map (default is "error"):

  golint -> warning
  varcheck -> warning
  structcheck -> warning
  deadcode -> warning
  gocyclo -> warning
  go-nyet -> warning
  errcheck -> warning

Flags:
  --help            Show help.
  --fast            Only run fast linters.
  -i, --install     Attempt to install all known linters.
  -u, --update      Pass -u to go tool when installing.
  -D, --disable=LINTER
                    List of linters to disable.
  -d, --debug       Display messages for failed linters, etc.
  -j, --concurrency=16
                    Number of concurrent linters to run.
  --exclude=REGEXP  Exclude messages matching this regular expression.
  --cyclo-over=10   Report functions with cyclomatic complexity over N (using
                    gocyclo).
  --sort=none       Sort output by any of none, path, line, column, severity,
                    message.
  -t, --tests       Include test files for linters that support this option
  --deadline=5s     Cancel linters if they have not completed within this
                    duration.
  --errors          Only show errors.
  --linter=NAME:COMMAND:PATTERN
                    Specify a linter.
  --message-overrides=LINTER:MESSAGE
                    Override message from linter. {message} will be expanded to
                    the original message.
  --severity=LINTER:SEVERITY
                    Map of linter severities.

Args:
  [<path>]  Directory to lint.
```

Additional linters can be configured via the command line:

```
$ gometalinter --linter='vet:go tool vet -printfuncs=Infof,Debugf,Warningf,Errorf {paths}:PATH:LINE:MESSAGE' .
stutter.go:22::error: Repeating defer a.Close() inside function duplicateDefer (defercheck)
stutter.go:21:15:warning: error return value not checked (defer a.Close()) (errcheck)
stutter.go:22:15:warning: error return value not checked (defer a.Close()) (errcheck)
stutter.go:27:6:warning: error return value not checked (doit()           // test for errcheck) (errcheck)
stutter.go:9::warning: unused global variable unusedGlobal (varcheck)
stutter.go:13::warning: unused struct field MyStruct.Unused (structcheck)
stutter.go:12:6:warning: exported type MyStruct should have comment or be unexported (golint)
stutter.go:16:6:warning: exported type PublicUndocumented should have comment or be unexported (deadcode)
```

