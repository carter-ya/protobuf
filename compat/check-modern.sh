#!/bin/sh
set -eu
repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
cd "$repo"
case "$(protoc --version)" in 'libprotoc 36.2') ;; *) echo 'Modern Go integration requires pinned protoc 36.2'; exit 1;; esac
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT HUP INT TERM
protoc -I . -I protobuf --include_imports --descriptor_set_out="$tmp/schema.pb" \
 compat/optional/reflection/sample.proto \
 compat/editions/edition2023/sample.proto compat/editions/edition2023/foreign_ext.proto \
 compat/editions/edition2024/sample.proto compat/editions/edition2026/sample.proto \
 google/protobuf/any.proto google/protobuf/wrappers.proto google/protobuf/duration.proto
python3 compat/optional/mappings/prepare.py "$tmp/mappings" --modern
cd compat/modern
GOGO_MODERN_DESCRIPTOR="$tmp/schema.pb" go test ./...
GOGO_MODERN_DESCRIPTOR="$tmp/schema.pb" go test -tags=purego ./...
GOGO_MODERN_DESCRIPTOR="$tmp/schema.pb" go test -race ./...
