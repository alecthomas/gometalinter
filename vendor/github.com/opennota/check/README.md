check [![License](http://img.shields.io/:license-gpl3-blue.svg)](http://www.gnu.org/licenses/gpl-3.0.html) [![Build Status](https://travis-ci.org/opennota/check.png?branch=master)](https://travis-ci.org/opennota/check)
=====

A set of utilities for checking Go sources.

## Installation

    $ go get github.com/opennota/check/cmd/aligncheck
    $ go get github.com/opennota/check/cmd/structcheck
    $ go get github.com/opennota/check/cmd/varcheck

## Usage

Find inefficiently packed structs.


```
$ aligncheck net/http
net/http: /usr/lib/go/src/net/http/server.go:123:6: struct conn could have size 160 (currently 168)
net/http: /usr/lib/go/src/net/http/server.go:315:6: struct response could have size 152 (currently 176)
net/http: /usr/lib/go/src/net/http/transfer.go:37:6: struct transferWriter could have size 96 (currently 112)
net/http: /usr/lib/go/src/net/http/transport.go:49:6: struct Transport could have size 136 (currently 144)
net/http: /usr/lib/go/src/net/http/transport.go:811:6: struct persistConn could have size 160 (currently 176)

```
For the visualisation of struct packing see http://golang-sizeof.tips/

Find unused struct fields.

```
$ structcheck --help
Usage of structcheck:
  -a    Count assignments only
  -e    Report exported fields
  -t    Load test files too

$ structcheck fmt
fmt: /usr/lib/go/src/fmt/format.go:47:2: fmt.fmtFlags.zero
fmt: /usr/lib/go/src/fmt/format.go:41:2: fmt.fmtFlags.minus
fmt: /usr/lib/go/src/fmt/format.go:42:2: fmt.fmtFlags.plus
fmt: /usr/lib/go/src/fmt/format.go:43:2: fmt.fmtFlags.sharp
fmt: /usr/lib/go/src/fmt/format.go:44:2: fmt.fmtFlags.space
fmt: /usr/lib/go/src/fmt/format.go:52:2: fmt.fmtFlags.plusV
fmt: /usr/lib/go/src/fmt/format.go:53:2: fmt.fmtFlags.sharpV
fmt: /usr/lib/go/src/fmt/format.go:39:2: fmt.fmtFlags.widPresent
fmt: /usr/lib/go/src/fmt/format.go:40:2: fmt.fmtFlags.precPresent
fmt: /usr/lib/go/src/fmt/format.go:45:2: fmt.fmtFlags.unicode
fmt: /usr/lib/go/src/fmt/format.go:46:2: fmt.fmtFlags.uniQuote
fmt: /usr/lib/go/src/fmt/print.go:110:2: fmt.pp.n
fmt: /usr/lib/go/src/fmt/scan.go:179:2: fmt.ssave.nlIsEnd
fmt: /usr/lib/go/src/fmt/scan.go:180:2: fmt.ssave.nlIsSpace
fmt: /usr/lib/go/src/fmt/scan.go:181:2: fmt.ssave.argLimit
fmt: /usr/lib/go/src/fmt/scan.go:182:2: fmt.ssave.limit
fmt: /usr/lib/go/src/fmt/scan.go:183:2: fmt.ssave.maxWid
```

Find unused global variables and constants.

```
$ varcheck --help
Usage of varcheck:
  -e=false: Report exported variables and constants

$ varcheck image/jpeg
image/jpeg: /usr/lib/go/src/image/jpeg/reader.go:74:2: adobeTransformYCbCr
image/jpeg: /usr/lib/go/src/image/jpeg/reader.go:75:2: adobeTransformYCbCrK
image/jpeg: /usr/lib/go/src/image/jpeg/writer.go:54:2: quantIndexLuminance
image/jpeg: /usr/lib/go/src/image/jpeg/writer.go:55:2: quantIndexChrominance
image/jpeg: /usr/lib/go/src/image/jpeg/writer.go:91:2: huffIndexLuminanceDC
image/jpeg: /usr/lib/go/src/image/jpeg/writer.go:92:2: huffIndexLuminanceAC
image/jpeg: /usr/lib/go/src/image/jpeg/writer.go:93:2: huffIndexChrominanceDC
image/jpeg: /usr/lib/go/src/image/jpeg/writer.go:94:2: huffIndexChrominanceAC
```

## Known limitations

structcheck doesn't handle embedded structs yet.
