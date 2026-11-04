#!/bin/sh
set -eu
repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
cd "$repo"
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT HUP INT TERM
go build -o "$tmp/driver" ./protoc-gen-gogo
go build -o "$tmp/fast-driver" ./protoc-gen-gogofast
mkdir -p "$tmp/reflection" "$tmp/fast"
protoc -I . -I protobuf --plugin=protoc-gen-test="$tmp/driver" --test_out=paths=source_relative:"$tmp/reflection" compat/optional/reflection/sample.proto
protoc -I . -I protobuf --plugin=protoc-gen-test="$tmp/fast-driver" --test_out=paths=source_relative:"$tmp/fast" compat/optional/fast/sample.proto
go run ./compat/compare-generated -- compat/optional/reflection/sample.pb.go "$tmp/reflection/compat/optional/reflection/sample.pb.go"
go run ./compat/compare-generated -- compat/optional/fast/sample.pb.go "$tmp/fast/compat/optional/fast/sample.pb.go"
go test ./compat/optional/...
