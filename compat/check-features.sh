#!/bin/sh
set -eu
repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
cd "$repo"
go test ./internal/...
python3 compat/test-merge-protocol.py
case "$(protoc --version)" in
  'libprotoc 36.2')
    tmp=$(mktemp -d)
    trap 'rm -rf "$tmp"' EXIT HUP INT TERM
    sh compat/regenerate-features.sh "$tmp/defaults.gen.go"
    diff -u internal/semantics/defaults.gen.go "$tmp/defaults.gen.go"
    go build -o "$tmp/gogo" ./protoc-gen-gogo
    protoc -I compat/testdata --plugin=protoc-gen-test="$tmp/gogo" --test_out="$tmp" edition2023.proto
    ;;
esac
