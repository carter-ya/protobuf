#!/bin/sh
set -eu
repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
cd "$repo"
case "$(protoc --version)" in 'libprotoc 36.2') ;; *) echo 'Editions oracle requires pinned protoc 36.2'; exit 0;; esac
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT HUP INT TERM
go build -o "$tmp/gogo" ./protoc-gen-gogo
go build -o "$tmp/fast" ./protoc-gen-gogofast
protoc -I . -I protobuf --plugin=protoc-gen-test="$tmp/gogo" --test_out=paths=source_relative:"$tmp" compat/editions/edition2023/imported.proto compat/editions/edition2023/sample.proto compat/editions/edition2023/inherited.proto compat/editions/edition2023/foreign_ext.proto
for file in imported sample inherited foreign_ext; do
 go run ./compat/compare-generated -- "compat/editions/edition2023/$file.pb.go" "$tmp/compat/editions/edition2023/$file.pb.go"
done
GOGO_EDITION_PROTOC="$(command -v protoc)" go test ./compat/editions/...
# Option imports need the compiler's pinned modern include tree, without
# replacing the legacy standard-proto fixtures used by old projects.
protoc -I . --plugin=protoc-gen-test="$tmp/gogo" --test_out=paths=source_relative:"$tmp" compat/editions/edition2024/sample.proto
go run ./compat/compare-generated -- compat/editions/edition2024/sample.pb.go "$tmp/compat/editions/edition2024/sample.pb.go"
# OPAQUE is now a supported opt-in API; its runtime behavior is qualified in
# the Go 1.23+ OPAQUE matrix, while this gate also runs on legacy Go 1.15.
sed 's/API_OPEN/API_OPAQUE/' compat/editions/edition2024/sample.proto > "$tmp/opaque.proto"
protoc -I "$tmp" -I . --plugin=protoc-gen-test="$tmp/gogo" --test_out=paths=source_relative:"$tmp" opaque.proto
grep -q 'xxx_hidden_Value' "$tmp/opaque.pb.go"
grep -q 'GoGoProtoPackageIsVersion18' "$tmp/opaque.pb.go"
grep -q 'ProtoReflect()' "$tmp/opaque.pb.go"
if protoc -I . --plugin=protoc-gen-test="$tmp/gogo" --test_out=default_api_level=API_UNKNOWN:"$tmp" compat/editions/edition2024/sample.proto > "$tmp/error" 2>&1; then
 echo 'unknown API level accepted' >&2; exit 1
fi
grep -q 'default_api_level supports' "$tmp/error"
protoc -I . --plugin=protoc-gen-test="$tmp/gogo" --test_out=paths=source_relative:"$tmp" compat/editions/edition2026/sample.proto
go run ./compat/compare-generated -- compat/editions/edition2026/sample.pb.go "$tmp/compat/editions/edition2026/sample.pb.go"
protoc -I . --include_imports --descriptor_set_out="$tmp/2026.pb" compat/editions/edition2026/sample.proto
case "$(go version)" in
 'go version go1.15.'*) echo 'Official 2026 oracle runs on Go 1.23+; old runtime/compiler checks completed' ;;
 *) (cd compat/oracle && GOGO_ORACLE_DESCRIPTOR="$tmp/2026.pb" go test ./...) ;;
esac
# Newly generated features must not silently link to the original runtime.
mkdir -p "$tmp/old"
git archive v1.3.2 | tar -xf - -C "$tmp/old"
for edition in 2023 2024 2026; do
 consumer="$tmp/consumer$edition"
 mkdir -p "$consumer"
 cp compat/editions/edition$edition/*.pb.go "$consumer/"
 cp compat/legacy/go.sum "$consumer/go.sum"
 cat > "$consumer/go.mod" <<MOD
module example.com/old-editions-runtime-check
go 1.15
require github.com/gogo/protobuf v1.3.2
replace github.com/gogo/protobuf => $tmp/old
MOD
 if (cd "$consumer" && go build ./...) > "$tmp/error" 2>&1; then
  echo "Edition $edition compiled with original runtime" >&2; exit 1
 fi
 case "$edition" in
  2023) required=14 ;; # The maintained fixture includes required initialization metadata.
  2026) required=6 ;;
  *) required=5 ;;
 esac
 grep -q "GoGoProtoPackageIsVersion$required" "$tmp/error"
done
