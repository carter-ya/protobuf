#!/bin/sh
set -eu
repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
: "${PROTOBUF_CONFORMANCE_SOURCE:?Set to the pinned 36.2 source directory}"
: "${PROTOBUF_CONFORMANCE_RUNNER:?Set to the pinned runner built by compat/conformance/build-runner.py}"
: "${CONFORMANCE_OUTPUT:?Set to a persistent output directory for reports and logs}"
python3 -m unittest discover -s "$repo/compat/conformance" -p "test_*.py"
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT HUP INT TERM
python3 "$repo/compat/conformance/prepare.py" "$PROTOBUF_CONFORMANCE_SOURCE" "$tmp/module"
python3 "$repo/compat/conformance/run.py" "$PROTOBUF_CONFORMANCE_RUNNER" "$tmp/module/testee" "$CONFORMANCE_OUTPUT/legacy"
python3 "$repo/compat/conformance/prepare.py" "$PROTOBUF_CONFORMANCE_SOURCE" "$tmp/standard" --merge-oneof-messages
python3 "$repo/compat/conformance/run.py" "$PROTOBUF_CONFORMANCE_RUNNER" "$tmp/standard/testee" "$CONFORMANCE_OUTPUT/standard" --mode standard
