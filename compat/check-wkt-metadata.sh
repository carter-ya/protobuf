#!/bin/sh
set -eu
repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
cd "$repo"
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT HUP INT TERM
sh compat/regenerate-wkt-metadata.sh "$tmp/generated"
for file in type api; do
 go run ./compat/compare-generated -- "types/$file.pb.go" "$tmp/generated/$file.pb.go"
done
mkdir -p "$tmp/old" "$tmp/consumer/types"
git archive v1.3.2 protobuf types | tar -xf - -C "$tmp/old"
protoc -I "$tmp/old/protobuf" --include_imports --descriptor_set_out="$tmp/old.pb" google/protobuf/type.proto google/protobuf/api.proto
protoc -I protobuf --include_imports --descriptor_set_out="$tmp/current.pb" google/protobuf/type.proto google/protobuf/api.proto
go run ./compat/wktmeta "$tmp/old.pb" "$tmp/current.pb"
for name in type api any source_context; do cp "$tmp/old/types/$name.pb.go" "$tmp/consumer/types/"; done
cp compat/wktlegacy/legacy_test.go "$tmp/consumer/"
cp go.sum "$tmp/consumer/go.sum"
cat > "$tmp/consumer/go.mod" <<MOD
module example.com/gogo-wkt-legacy
go 1.15
require github.com/gogo/protobuf v1.3.2
replace github.com/gogo/protobuf => $repo
MOD
(cd "$tmp/consumer" && go test ./...)
go test ./types ./jsonpb/standard
