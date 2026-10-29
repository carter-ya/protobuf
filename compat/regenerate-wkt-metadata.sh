#!/bin/sh
set -eu
repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
out=${1:-"$repo/types"}
mkdir -p "$out"
out=$(CDPATH= cd -- "$out" && pwd)
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT HUP INT TERM
cd "$repo"
go build -o "$tmp/gogotypes" ./protoc-gen-gogotypes
protoc -I protobuf --plugin=protoc-gen-gogotypes="$tmp/gogotypes" \
 --gogotypes_out=paths=source_relative:"$tmp" google/protobuf/type.proto google/protobuf/api.proto
cp "$tmp/google/protobuf/type.pb.go" "$out/type.pb.go"
cp "$tmp/google/protobuf/api.pb.go" "$out/api.pb.go"
