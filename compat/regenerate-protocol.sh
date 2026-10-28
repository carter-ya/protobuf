#!/bin/sh
set -eu
repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
cd "$repo"
# Override these to bootstrap with the frozen v1.3.2 binaries. These proto2
# protocol definitions can be generated before optional support is enabled.
: "${GOGO_PLUGIN:=protoc-gen-gogo}"
: "${GOSTRING_PLUGIN:=protoc-gen-gostring}"
protoc -I protobuf/google/protobuf \
  --plugin=protoc-gen-gogo="$GOGO_PLUGIN" \
  --gogo_out=protoc-gen-gogo/descriptor protobuf/google/protobuf/descriptor.proto
protoc -I protobuf/google/protobuf \
  --plugin=protoc-gen-gostring="$GOSTRING_PLUGIN" \
  --gostring_out=protoc-gen-gogo/descriptor protobuf/google/protobuf/descriptor.proto
protoc -I protobuf/google/protobuf/compiler -I protobuf \
  --plugin=protoc-gen-gogo="$GOGO_PLUGIN" \
  --gogo_out=Mgoogle/protobuf/descriptor.proto=github.com/gogo/protobuf/protoc-gen-gogo/descriptor:protoc-gen-gogo/plugin \
  protobuf/google/protobuf/compiler/plugin.proto
