#!/bin/sh
set -eu
repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
: "${PROTOBUF_CONFORMANCE_SOURCE:?Set to pinned 36.2 source}"
: "${PROTOBUF_CONFORMANCE_RUNNER:?Set to pinned C++ runner with wire_oracle}"
: "${BINARY_FUZZ_OUTPUT:?Set to a fresh persistent result directory}"
output="$BINARY_FUZZ_OUTPUT"
mkdir -p "$output"
output=$(CDPATH= cd -- "$output" && pwd)
python3 "$repo/compat/qualify/binary/prepare.py" "$PROTOBUF_CONFORMANCE_SOURCE" "$output/modules"
export GOGO_WIRE_DESCRIPTOR="$output/modules/schema.pb"
for backend in gogo gogofast gogofaster gogoslick gogotypes; do
  (
    cd "$output/modules/$backend"
    go test ./... > "$output/$backend-seeds.log" 2>&1
    go test -tags purego ./... > "$output/$backend-purego.log" 2>&1
    go test -race ./... > "$output/$backend-race.log" 2>&1
    GOMAXPROCS=2 go test -run '^$' -fuzz '^FuzzWire$' -fuzztime="${BINARY_FUZZ_TIME:-20s}" -parallel=2 > "$output/$backend-fuzz.log" 2>&1
    GOMAXPROCS=2 go test -tags purego -run '^$' -fuzz '^FuzzWire$' -fuzztime="${BINARY_FUZZ_TIME:-20s}" -parallel=2 > "$output/$backend-purego-fuzz.log" 2>&1
  )
done
