# Go Meta Linter [![Build Status](https://travis-ci.org/alecthomas/gometalinter.png)](https://travis-ci.org/alecthomas/gometalinter)

The number of tools for statically checking Go source for errors and warnings
is impressive.

This is a tool that concurrently runs a whole bunch of those linters and
normalises their output to a standard format:

    <file>:<line>:[<column>]: <message> (<linter>)

eg.

    stutter.go:9::warning: unused global variable unusedGlobal (varcheck)
    stutter.go:12:6:warning: exported type MyStruct should have comment or be unexported (golint)

It is intended for use with editor/IDE integration.

## Editor integration

- [SublimeLinter plugin](https://github.com/alecthomas/SublimeLinter-contrib-gometalinter).
- There is a ticket open for [vim-go](https://github.com/fatih/vim-go/issues/320), opened Feb 2015.

## Supported linters

- [gofmt -s](https://golang.org/cmd/gofmt/) - Checks if the code is properly formatted and could not be further simplified.
- [go vet](https://golang.org/cmd/vet/) - Reports potential errors that otherwise compile.
- [go vet --shadow](https://golang.org/cmd/vet/#hdr-Shadowed_variables) - Reports variables that may have been unintentionally shadowed.
- [gotype](https://golang.org/x/tools/cmd/gotype) - Syntactic and semantic analysis similar to the Go compiler.
- [goimports](https://godoc.org/golang.org/x/tools/cmd/goimports) - Checks missing or unreferenced package imports.
- [deadcode](https://github.com/remyoudompheng/go-misc/tree/master/deadcode) - Finds unused code.
- [gocyclo](https://github.com/alecthomas/gocyclo) - Computes the cyclomatic complexity of functions.
- [golint](https://github.com/golang/lint) - Google's (mostly stylistic) linter.
- [defercheck](https://github.com/opennota/check) - Checks for duplicate defer calls.
- [varcheck](https://github.com/opennota/check) - Find unused global variables and constants.
- [structcheck](https://github.com/opennota/check) - Find unused struct fields.
- [aligncheck](https://github.com/opennota/check) - Warn about un-optimally aligned structures.
- [errcheck](https://github.com/alecthomas/errcheck) - Check that error return values are used.
- [dupl](https://github.com/mibk/dupl) - Reports potentially duplicated code.
- [ineffassign](https://github.com/gordonklaus/ineffassign/blob/master/list) - Detect when assignments to *existing* variables are not used.

Disabled by default (enable with `--enable=<linter>`):

- [testify](https://github.com/stretchr/testify) - Show location of failed testify assertions (disabled by default).
- [test](http://golang.org/pkg/testing/) - Show location of test failures from the stdlib testing module (disabled by default).

Additional linters can be added through the command line with `--linter=NAME:COMMAND:PATTERN` (see [below](#details)).

## Quickstart

Install gometalinter:

```
$ go get github.com/alecthomas/gometalinter
```

Install all known linters:

```
$ gometalinter --install --update
Installing structcheck -> go get github.com/opennota/check/cmd/structcheck
Installing aligncheck -> go get github.com/opennota/check/cmd/aligncheck
Installing deadcode -> go get github.com/remyoudompheng/go-misc/deadcode
Installing gocyclo -> go get github.com/alecthomas/gocyclo
Installing ineffassign -> go get github.com/gordonklaus/ineffassign
Installing dupl -> go get github.com/mibk/dupl
Installing golint -> go get github.com/golang/lint/golint
Installing gotype -> go get golang.org/x/tools/cmd/gotype
Installing goimports -> go get golang.org/x/tools/cmd/goimports
Installing errcheck -> go get github.com/alecthomas/errcheck
Installing defercheck -> go get github.com/opennota/check/cmd/defercheck
Installing varcheck -> go get github.com/opennota/check/cmd/varcheck
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

## FAQ

### How do I make `gometalinter` work with Go 1.5 vendoring?

`gometalinter` has no specific support for vendor paths, however if the underlying tools support it then it should Just Work™. Ensure that all of the linters are up to date and built with Go 1.5 (`gometalinter --install --update --force`) then run `GO_VENDOR=1 gometalinter .`. That should be it.

### Why does `gometalinter --install` install forks of gocyclo and errcheck?

I forked `gocyclo` because the upstream behaviour is to recursively check all
subdirectories even when just a single directory is specified. This made it
unusably slow when vendoring. The recursive behaviour can be achieved with
gometalinter by explicitly specifying `<path>/...`. There is a
[pull request](https://github.com/fzipp/gocyclo/pull/1) open.

`errcheck` was forked from [an old version](https://github.com/kisielk/errcheck/commit/0ba3e8228e4772238bee75d33c4cb0c3fb2a432c) that was fast.
Upstream subsequently switched to `go/loader` and became [slow](https://github.com/kisielk/errcheck/issues/56)
enough to not be usable in an interactive environment. There doesn't seem to be any
functional difference, and in lieu of any interest from upstream in fixing the issue,
the fork remains.

### Gometalinter is not working

That's more of a statement than a question, but okay.

Sometimes gometalinter will not report issues that you think it should. There
are three things to try in that case:

#### 1. Update to the latest build of gometalinter and all linters

    go get -u github.com/alecthomas/gometalinter
    gometalinter --install --update

If you're lucky, this will fix the problem.

#### 2. Analyse the debug output

If that doesn't help, the problem may be elsewhere (in no particular order):

1. Upstream linter has changed its output or semantics.
2. gometalinter is not invoking the tool correctly.
3. gometalinter regular expression matches are not correct for a linter.
4. Linter is exceeding the deadline.

To find out what's going on run in debug mode:

    gometalinter --debug

This will show all output from the linters and should indicate why it is
failing.

#### 3. Report an issue.

Failing all else, if the problem looks like a bug please file an issue and
include the output of `gometalinter --debug`.

## Details

```
$ gometalinter --help
usage: gometalinter [<flags>] [<path>]

Aggregate and normalise the output of a whole bunch of Go linters.

Default linters:

  gotype (golang.org/x/tools/cmd/gotype)
      gotype {tests=-a} {path}
      :PATH:LINE:COL:MESSAGE
  goimports (golang.org/x/tools/cmd/goimports)
      goimports -d {path}
      :^diff\s(?P<path>\S+)\s.+\s.+\s.+\s@@\s-(?P<line>\d+)[\S\s]+import
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
  vet ()
      go vet {path}
      :PATH:LINE:MESSAGE
  vet --shadow ()
      go vet --shadow {path}
      :PATH:LINE:MESSAGE
  gofmt ()
      gofmt -s -d -e {path}
      :^diff\s(?P<path>\S+)\s.+\s.+\s.+\s@@\s-(?P<line>\d+)`

Severity override map (default is "error"):

  golint -> warning
  varcheck -> warning
  structcheck -> warning
  deadcode -> warning
  gocyclo -> warning
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

