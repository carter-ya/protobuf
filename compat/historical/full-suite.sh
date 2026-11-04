#!/bin/sh
# Run all maintained root tests with a compiler that can parse current metadata,
# in the GOPATH layout required by historical integration fixtures.
set -eu
repo=$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)
compiler=${1:?pass the pinned current protoc path}
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT HUP INT TERM
export GOBIN="$tmp/bin"
mkdir -p "$GOBIN"
export PATH="$GOBIN:$(dirname -- "$compiler"):$PATH"
cd "$repo"
# The nested compatibility script owns and removes its own tool directory.
# Keep separate drivers alive for the later purego integration tests.
go install ./protoc-gen-gogo ./protoc-gen-gogofast ./protoc-gen-gogofaster ./protoc-gen-gogoslick ./protoc-gen-gogotypes ./protoc-min-version
sh compat/check.sh
suite="$tmp/gopath/src/github.com/gogo/protobuf"
mkdir -p "$suite"
tar --exclude=.git -cf - . | tar -xf - -C "$suite"
cp compat/golden/multi/*.pb.go "$suite/protoc-gen-gogo/testdata/multi/"
cd "$suite"
# Preserve the original purego exclusions for upstream issues #447/#448;
# modern targeted default/purego/race gates remain separately mandatory.
packages=$(go list ./... | grep -Ev '/jsonpb|/test/casttype/|/test/oneof/combos/')
go test -tags purego $packages
go vet ./...
# The archived shadow analyzer is qualified with the minimum supported Go.
# Resolve it through the root's exact x/tools pin, never go get @latest.
case "$(go version)" in
 *'go1.15.'*)
  export GOBIN="$tmp/bin"
  mkdir -p "$GOBIN"
  go install golang.org/x/tools/go/analysis/passes/shadow/cmd/shadow
  go version -m "$GOBIN/shadow"
  go vet -vettool="$GOBIN/shadow" ./...
 ;;
esac
