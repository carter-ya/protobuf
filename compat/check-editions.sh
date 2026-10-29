#!/bin/sh
set -eu
repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
cd "$repo"
case "$(protoc --version)" in 'libprotoc 36.2') ;; *) echo 'Editions oracle requires pinned protoc 36.2'; exit 0;; esac
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT HUP INT TERM
go build -o "$tmp/driver" ./compat/testdata/editions-driver
go build -o "$tmp/gogo" ./protoc-gen-gogo
go build -o "$tmp/fast" ./protoc-gen-gogofast
protoc -I . -I protobuf --plugin=protoc-gen-test="$tmp/driver" --test_out=paths=source_relative:"$tmp" compat/editions/edition2023/imported.proto compat/editions/edition2023/sample.proto compat/editions/edition2023/inherited.proto compat/editions/edition2023/foreign_ext.proto
for file in imported sample inherited foreign_ext; do
 go run ./compat/compare-generated -- "compat/editions/edition2023/$file.pb.go" "$tmp/compat/editions/edition2023/$file.pb.go"
done
GOGO_EDITION_PROTOC="$(command -v protoc)" go test ./compat/editions/...
# Capabilities remain disabled until the production entrypoint layer.
if protoc -I . -I protobuf --plugin=protoc-gen-test="$tmp/gogo" --test_out="$tmp" compat/editions/edition2023/sample.proto > "$tmp/error" 2>&1; then
 echo 'Editions unexpectedly advertised by production generator' >&2; exit 1
fi
grep -q 'support editions' "$tmp/error"
if protoc -I . -I protobuf --plugin=protoc-gen-test="$tmp/fast" --test_out="$tmp" compat/editions/edition2023/sample.proto > "$tmp/error" 2>&1; then
 echo 'unsupported fast Editions combination accepted' >&2; exit 1
fi
grep -q 'gogo extension.*not yet supported with Editions' "$tmp/error"
