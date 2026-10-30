#!/bin/sh
set -eu
repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT HUP INT TERM
python3 "$repo/compat/optional/mappings/prepare.py" "$tmp/consumers"
# A runtime that already supports Editions fast codecs still lacks the
# named-type-safe optional runtime operations introduced with Version10.
mkdir -p "$tmp/old"
(cd "$repo" && git archive 05e3bc0ed27d3862386062a4f563460c5fe1e381) | tar -xf - -C "$tmp/old"
consumer="$tmp/consumers/cast-gogo"
cat > "$consumer/go.mod" <<MOD
module example.com/optional-mappings
go 1.15
require github.com/gogo/protobuf v1.3.2
replace github.com/gogo/protobuf => $tmp/old
MOD
if (cd "$consumer" && go build ./...) > "$tmp/error" 2>&1; then
 echo 'optional mapping code compiled against the incomplete runtime' >&2; exit 1
fi
grep -q GoGoProtoPackageIsVersion10 "$tmp/error"
