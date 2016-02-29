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
- [vim-go](https://github.com/fatih/vim-go) with the `:GoMetaLinter` command.
- [syntastic (vim)](https://github.com/scrooloose/syntastic/wiki/Go:---gometalinter) `let g:syntastic_go_checkers = ['gometalinter']`
- [Atom linter plugin](https://atom.io/packages/gometalinter-linter).
- [Emacs Flycheck checker](https://github.com/favadi/flycheck-gometalinter).

## Supported linters

- [go vet](https://golang.org/cmd/vet/) - Reports potential errors that otherwise compile.
- [go vet --shadow](https://golang.org/cmd/vet/#hdr-Shadowed_variables) - Reports variables that may have been unintentionally shadowed.
- [gotype](https://golang.org/x/tools/cmd/gotype) - Syntactic and semantic analysis similar to the Go compiler.
- [deadcode](https://github.com/remyoudompheng/go-misc/tree/master/deadcode) - Finds unused code.
- [gocyclo](https://github.com/alecthomas/gocyclo) - Computes the cyclomatic complexity of functions.
- [golint](https://github.com/golang/lint) - Google's (mostly stylistic) linter.
- [varcheck](https://github.com/opennota/check) - Find unused global variables and constants.
- [structcheck](https://github.com/opennota/check) - Find unused struct fields.
- [aligncheck](https://github.com/opennota/check) - Warn about un-optimally aligned structures.
- [errcheck](https://github.com/kisielk/errcheck) - Check that error return values are used.
- [dupl](https://github.com/mibk/dupl) - Reports potentially duplicated code.
- [ineffassign](https://github.com/gordonklaus/ineffassign/blob/master/list) - Detect when assignments to *existing* variables are not used.
- [interfacer](https://github.com/mvdan/interfacer) - Suggest narrower interfaces that can be used.
- [unconvert](https://github.com/mdempsky/unconvert) - Detect redundant type conversions.
- [goconst](https://github.com/jgautheron/goconst) - Finds repeated strings that could be replaced by a constant.

Disabled by default (enable with `--enable=<linter>`):

- [testify](https://github.com/stretchr/testify) - Show location of failed testify assertions.
- [test](http://golang.org/pkg/testing/) - Show location of test failures from the stdlib testing module.
- [gofmt -s](https://golang.org/cmd/gofmt/) - Checks if the code is properly formatted and could not be further simplified.
- [goimports](https://godoc.org/golang.org/x/tools/cmd/goimports) - Checks missing or unreferenced package imports.
- [lll](https://github.com/walle/lll) - Report long lines (see `--line-length=N`).

Additional linters can be added through the command line with `--linter=NAME:COMMAND:PATTERN` (see [below](#details)).

## Quickstart

Install gometalinter:

```
$ go get github.com/alecthomas/gometalinter
```

Install all known linters:

```
$ gometalinter --install --update
Installing:
  structcheck
  aligncheck
  deadcode
  gocyclo
  ineffassign
  dupl
  golint
  gotype
  goimports
  errcheck
  varcheck
  interfacer
  goconst
```

Run it:

```
$ cd example
$ gometalinter ./...
stutter.go:13::warning: unused struct field MyStruct.Unused (structcheck)
stutter.go:9::warning: unused global variable unusedGlobal (varcheck)
stutter.go:12:6:warning: exported type MyStruct should have comment or be unexported (golint)
stutter.go:16:6:warning: exported type PublicUndocumented should have comment or be unexported (golint)
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
format. Note that this can be *very* slow, and you may need to increase the linter `--deadline` to allow linters to complete.

## FAQ

### Exit status

gometalinter sets two bits of the exit status to indicate different issues:

| Bit | Meaning
|-----|----------
| 0   | A linter generated an issue.
| 1   | An underlying error occurred; eg. a linter failed to execute. In this situation a warning will also be displayed.

eg. linter only = 1, underlying only = 2, linter + underlying = 3

### How do I make `gometalinter` work with Go 1.5 vendoring?

`gometalinter` has no specific support for vendor paths, however if the
underlying tools support it then it should Just Work™. Ensure that all
of the linters are up to date and built with Go 1.5
(`gometalinter --install --update --force`) then run
`GO15VENDOREXPERIMENT=1 gometalinter .`. That should be it.

### Why does `gometalinter --install` install a fork of gocyclo?

I forked `gocyclo` because the upstream behaviour is to recursively check all
subdirectories even when just a single directory is specified. This made it
unusably slow when vendoring. The recursive behaviour can be achieved with
gometalinter by explicitly specifying `<path>/...`. There is a
[pull request](https://github.com/fzipp/gocyclo/pull/1) open.

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
usage: gometalinter [<flags>] [<path>...]

Aggregate and normalise the output of a whole bunch of Go linters.

Default linters:

gofmt
      gofmt -l -s ./*.go
      :^(?P<path>[^\n]+)$
gotype  (golang.org/x/tools/cmd/gotype)
      gotype -e {tests=-a} .
      :PATH:LINE:COL:MESSAGE
goimports (golang.org/x/tools/cmd/goimports)
      goimports -l ./*.go
      :^(?P<path>[^\n]+)$
testify
      go test
      :Location:\s+(?P<path>[^:]+):(?P<line>\d+)$\s+Error:\s+(?P<message>[^\n]+)
test
      go test
      :^--- FAIL: .*$\s+(?P<path>[^:]+):(?P<line>\d+): (?P<message>.*)$
dupl  (github.com/mibk/dupl)
      dupl -plumbing -threshold {duplthreshold} ./*.go
      :^(?P<path>[^\s][^:]+?\.go):(?P<line>\d+)-\d+:\s*(?P<message>.*)$
golint  (github.com/golang/lint/golint)
      golint -min_confidence {min_confidence} .
      :PATH:LINE:COL:MESSAGE
structcheck  (github.com/opennota/check/cmd/structcheck)
      structcheck {tests=-t} .
      :^(?:[^:]+: )?(?P<path>[^:]+):(?P<line>\d+):(?P<col>\d+):\s*(?P<message>.+)$
aligncheck  (github.com/opennota/check/cmd/aligncheck)
      aligncheck .
      :^(?:[^:]+: )?(?P<path>[^:]+):(?P<line>\d+):(?P<col>\d+):\s*(?P<message>.+)$
gocyclo  (github.com/alecthomas/gocyclo)
      gocyclo -over {mincyclo} .
      :^(?P<cyclo>\d+)\s+\S+\s(?P<function>\S+)\s+(?P<path>[^:]+):(?P<line>\d+):(\d+)$
vet
      go tool vet ./*.go
      :PATH:LINE:MESSAGE
errcheck  (github.com/alecthomas/errcheck)
      errcheck .
      :^(?P<path>[^:]+):(?P<line>\d+):(?P<col>\d+)\t(?P<message>.*)$
ineffassign  (github.com/gordonklaus/ineffassign)
      ineffassign -n .
      :PATH:LINE:COL:MESSAGE
vetshadow
      go tool vet --shadow ./*.go
      :PATH:LINE:MESSAGE
varcheck  (github.com/opennota/check/cmd/varcheck)
      varcheck .
      :^(?:[^:]+: )?(?P<path>[^:]+):(?P<line>\d+):(?P<col>\d+):\s*(?P<message>\w+)$
deadcode  (github.com/tsenart/deadcode)
      deadcode .
      :^deadcode: (?P<path>[^:]+):(?P<line>\d+):(?P<col>\d+):\s*(?P<message>.*)$
interfacer  (github.com/mvdan/interfacer/cmd/interfacer)
      interfacer ./
      :PATH:LINE:MESSAGE
goconst  (github.com/jgautheron/goconst/cmd/goconst)
      goconst ./
      :PATH:LINE:COL:MESSAGE

Severity override map (default is "warning"):

gotype -> error
test -> error
testify -> error
vet -> error

Flags:
      --help                Show context-sensitive help (also try --help-long
                            and --help-man).
      --fast                Only run fast linters.
  -i, --install             Attempt to install all known linters.
  -u, --update              Pass -u to go tool when installing.
  -f, --force               Pass -f to go tool when installing.
  -d, --debug               Display messages for failed linters, etc.
  -j, --concurrency=16      Number of concurrent linters to run.
  -e, --exclude=REGEXP      Exclude messages matching these regular expressions.
      --cyclo-over=10       Report functions with cyclomatic complexity over N
                            (using gocyclo).
      --line-length=80      Report lines longer than N (using lll).
      --min-confidence=.80  Minimum confidence interval to pass to golint.
      --min-occurrences=3   Minimum occurrences to pass to goconst.
      --dupl-threshold=50   Minimum token sequence as a clone for dupl.
      --sort=none           Sort output by any of none, path, line, column,
                            severity, message, linter.
  -t, --tests               Include test files for linters that support this
                            option
      --deadline=5s         Cancel linters if they have not completed within
                            this duration.
      --errors              Only show errors.
      --json                Generate structured JSON rather than standard
                            line-based output.
  -D, --disable=LINTER      List of linters to disable (testify,test).
  -E, --enable=LINTER       Enable previously disabled linters.
      --linter=NAME:COMMAND:PATTERN
                            Specify a linter.
      --message-overrides=LINTER:MESSAGE
                            Override message from linter. {message} will be
                            expanded to the original message.
      --severity=LINTER:SEVERITY
                            Map of linter severities.
      --disable-all         Disable all linters.

Args:
  [<path>]  Directory to lint. Defaults to ".". <path>/... will recurse.
```

Additional linters can be configured via the command line:

```
$ gometalinter --linter='vet:go tool vet -printfuncs=Infof,Debugf,Warningf,Errorf {paths}:PATH:LINE:MESSAGE' .
stutter.go:21:15:warning: error return value not checked (defer a.Close()) (errcheck)
stutter.go:22:15:warning: error return value not checked (defer a.Close()) (errcheck)
stutter.go:27:6:warning: error return value not checked (doit()           // test for errcheck) (errcheck)
stutter.go:9::warning: unused global variable unusedGlobal (varcheck)
stutter.go:13::warning: unused struct field MyStruct.Unused (structcheck)
stutter.go:12:6:warning: exported type MyStruct should have comment or be unexported (golint)
stutter.go:16:6:warning: exported type PublicUndocumented should have comment or be unexported (deadcode)
```
