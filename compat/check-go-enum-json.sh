#!/bin/sh
set -eu
repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
case "$(protoc --version)" in
 'libprotoc 36.2'|'libprotoc 37.0-rc1') ;;
 *) echo 'Go enum features require a qualified Editions compiler'; exit 0 ;;
esac
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT HUP INT TERM
python3 "$repo/compat/go-enum-json/prepare.py" "$tmp/consumer"

(cd "$repo" && go build -o "$tmp/gogo" ./protoc-gen-gogo)
mkdir -p "$tmp/schema"
protoc -I "$repo/protobuf" --plugin=protoc-gen-test="$tmp/gogo"  --test_out=paths=source_relative,Mgoogle/protobuf/go_features.proto=github.com/gogo/protobuf/gofeatures,Mgoogle/protobuf/descriptor.proto=github.com/gogo/protobuf/protoc-gen-gogo/descriptor:"$tmp/schema"  google/protobuf/go_features.proto
(cd "$repo" && go run ./compat/compare-generated -- gofeatures/go_features.pb.go "$tmp/schema/google/protobuf/go_features.pb.go")

# An application can still choose its own binding for ordinary feature imports.
mkdir -p "$tmp/mapped"
protoc -I "$tmp/consumer/gogo-2023-default" --plugin=protoc-gen-test="$tmp/gogo"  --test_out=paths=source_relative,Mgoogle/protobuf/go_features.proto=example.com/custom/features:"$tmp/mapped" enums.proto
grep -q '"example.com/custom/features"' "$tmp/mapped/enums.pb.go"
