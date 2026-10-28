#!/bin/sh
set -eu
repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT HUP INT TERM
cd "$repo"
export GOBIN="$tmp/bin"
mkdir -p "$GOBIN" "$tmp/reflection" "$tmp/fast"
go install ./protoc-gen-gogo ./protoc-gen-gogofaster ./protoc-min-version ./compat/compare-generated
export PATH="$GOBIN:$PATH"
# Keep protoc constant when comparing generator output.
protoc -I compat/schema --gogo_out=paths=source_relative:"$tmp/reflection" compat/schema/legacy.proto
protoc -I compat/schema --gogofaster_out=paths=source_relative:"$tmp/fast" compat/schema/legacy_fast.proto
compare-generated compat/legacy/reflection/legacy.pb.go "$tmp/reflection/legacy.pb.go"
compare-generated compat/legacy/fast/legacy_fast.pb.go "$tmp/fast/legacy_fast.pb.go"
(cd compat/legacy && go test ./...)
# Several historical integration tests resolve imports through a GOPATH-shaped
# relative path. Run them in a disposable copy with that layout, without
# rewriting those tests or allowing generated output to dirty the worktree.
suite="$tmp/gopath/src/github.com/gogo/protobuf"
mkdir -p "$suite"
tar --exclude=.git -cf - . | tar -xf - -C "$suite"
cp compat/golden/multi/*.pb.go "$suite/protoc-gen-gogo/testdata/multi/"
(cd "$suite" && go test ./...)
