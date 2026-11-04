#!/bin/sh
# Valid legacy compiler requests retain backend defaults; Editions metadata is
# generated and checked separately using the pinned current compiler.
set -eu
repo=$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)
compiler=${1:?pass the historical protoc path}
version=$("$compiler" --version)
echo "Historical compiler: $version"
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT HUP INT TERM
export GOBIN="$tmp/bin"
mkdir -p "$GOBIN"
cd "$repo"
go install ./protoc-gen-gogo ./protoc-gen-gogofast ./protoc-gen-gogofaster ./protoc-gen-gogoslick ./protoc-gen-gogotypes
syntaxes='proto2 proto3'
case "$version" in 'libprotoc 2.'*) syntaxes='proto2';; esac
for syntax in $syntaxes; do
 for backend in gogo gogofast gogofaster gogoslick gogotypes; do
  consumer="$tmp/$syntax-$backend"
  mkdir -p "$consumer/pb"
  cp "compat/historical/$syntax.proto" "$consumer/schema.proto"
  cp compat/historical/consumer_test.go.txt "$consumer/consumer_test.go"
  cat > "$consumer/go.mod" <<MOD
module example.com/historical-consumer

go 1.15

require github.com/gogo/protobuf v1.4.0
replace github.com/gogo/protobuf => $repo
MOD
  (cd "$consumer" && "$compiler" -I . --plugin="protoc-gen-test=$GOBIN/protoc-gen-$backend" --test_out="paths=source_relative:$consumer/pb" schema.proto)
  (cd "$consumer" && go test -mod=mod -count=1 ./... && go test -mod=mod -count=1 -tags purego ./...)
 done
done
