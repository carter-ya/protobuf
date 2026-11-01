#!/bin/sh
set -eu
repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
cd "$repo"
case "$(protoc --version)" in 'libprotoc 36.2') ;; *) echo 'Live reflection fixture regeneration requires protoc 36.2'; exit 1;; esac
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT HUP INT TERM
go build -o "$tmp/protoc-gen-gogo" ./protoc-gen-gogo
go build -o "$tmp/compare-generated" ./compat/compare-generated
for schema in reflect/internal/testpb/*.proto; do
  policy=,merge_required_fields=true,enforce_closed_enums=true
  case "$schema" in */unsupported.proto) policy=,Mgoogle/protobuf/timestamp.proto=github.com/gogo/protobuf/types;; esac
  protoc -I . -I protobuf --plugin=protoc-gen-gogo="$tmp/protoc-gen-gogo" \
    --gogo_out=paths=source_relative$policy:"$tmp" "$schema"
  "$tmp/compare-generated" "${schema%.proto}.pb.go" "$tmp/${schema%.proto}.pb.go"
done
cd reflect
go test ./...
go test -tags=purego ./...
go test -race ./...
