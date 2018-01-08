#!/bin/sh
set -e

TAR_FILE="/tmp/gometalinter.tar.gz"
RELEASES_URL="https://github.com/gometalinter/gometalinter/releases"
test -z "$TMPDIR" && TMPDIR="$(mktemp -d)"

last_version() {
  curl -sL -o /dev/null -w %{url_effective} "$RELEASES_URL/latest" |
    rev |
    cut -f1 -d'/'|
    rev
}

download() {
  test -z "$VERSION" && VERSION="$(last_version)"
  test -z "$VERSION" && {
    echo "Unable to get gometalinter version." >&2
    exit 1
  }
  rm -f "$TAR_FILE"
  curl -s -L -o "$TAR_FILE" \
    "$RELEASES_URL/download/$VERSION/gometalinter_$(uname -s)_$(uname -m).tar.gz"
}

download
tar -xf "$TAR_FILE" -C "$TMPDIR"
"${TMPDIR}/gometalinter" "$@"
