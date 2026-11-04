#!/bin/sh
set -eu
repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
cd "$repo"
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT HUP INT TERM
export GOBIN="$tmp/bin"
mkdir -p "$GOBIN"
go install ./protoc-gen-gogo ./protoc-gen-gofast ./protoc-gen-gogofast ./protoc-gen-gogofaster ./protoc-gen-gogoslick ./protoc-gen-gogotypes ./protoc-gen-gostring
export PATH="$GOBIN:$PATH"
# Compile and execute the presence tests separately for each public generator,
# avoiding duplicate registrations of the same proto name in one process.
for generator in gogo gogofast gogofaster gogoslick gogotypes; do
  suite="$tmp/$generator"
  mkdir -p "$suite"
  tar --exclude=.git -cf - . | tar -xf - -C "$suite"
  (cd "$suite" && protoc -I . -I protobuf --"${generator}_out"=paths=source_relative:. compat/optional/fast/sample.proto && go test ./compat/optional/...)
done

# Compile auxiliary GoString output alongside a base without that method.
suite="$tmp/gogotypes"
sed 's/gogoproto.gostring_all) = true/gogoproto.gostring_all) = false/' compat/optional/fast/sample.proto > "$suite/compat/optional/fast/sample.proto"
(cd "$suite" && protoc -I . -I protobuf --gogo_out=paths=source_relative:. --gostring_out=paths=source_relative:. compat/optional/fast/sample.proto && go test ./compat/optional/...)

# Explicitly unsupported extension/runtime combinations fail at generation.
mkdir -p "$tmp/negative"
sed 's/optional int32 value = 1;/optional int32 value = 1 [(gogoproto.nullable) = false];/' compat/optional/reflection/sample.proto > "$tmp/negative/nullable.proto"
if protoc -I "$tmp/negative" -I . -I protobuf --gogo_out="$tmp/negative" nullable.proto > "$tmp/negative/error" 2>&1; then
  echo 'nullable=false unexpectedly accepted' >&2; exit 1
fi
grep -q 'nullable=false cannot preserve proto3 optional presence' "$tmp/negative/error"
sed 's/optional Child child = 17;/optional Child child = 17 [(gogoproto.casttype) = "CustomChild"];/' compat/optional/reflection/sample.proto > "$tmp/negative/custom.proto"
if protoc -I "$tmp/negative" -I . -I protobuf --gogo_out="$tmp/negative" custom.proto > "$tmp/negative/error" 2>&1; then
  echo 'unsupported casttype unexpectedly accepted' >&2; exit 1
fi
grep -q 'extension combination is not yet supported' "$tmp/negative/error"
if protoc -I . -I protobuf --gofast_out="$tmp/negative" compat/optional/reflection/sample.proto > "$tmp/negative/error" 2>&1; then
  echo 'alternate runtime unexpectedly accepted' >&2; exit 1
fi
grep -q 'requires the gogo runtime' "$tmp/negative/error"

# New feature code must not compile against the unmodified old runtime.
mkdir -p "$tmp/old" "$tmp/consumer"
git archive v1.3.2 | tar -xf - -C "$tmp/old"
cp compat/optional/fast/sample.pb.go "$tmp/consumer/sample.pb.go"
cp compat/legacy/go.sum "$tmp/consumer/go.sum"
cat > "$tmp/consumer/go.mod" <<EOF
module example.com/old-runtime-check
go 1.15
require github.com/gogo/protobuf v1.3.2
replace github.com/gogo/protobuf => $tmp/old
EOF
if (cd "$tmp/consumer" && go build ./...) > "$tmp/negative/error" 2>&1; then
  echo 'optional code compiled with an old runtime' >&2; exit 1
fi
grep -q 'GoGoProtoPackageIsVersion4' "$tmp/negative/error"
