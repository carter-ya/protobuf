#!/bin/sh
set -eu
repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT HUP INT TERM
python3 "$repo/compat/legacy-enums/prepare.py" "$tmp/map"
python3 "$repo/compat/legacy-enums/prepare.py" "$tmp/bytes" --byte-extensions

# Filter metadata must not silently compile against a pre-policy runtime.
introduction=$(git -C "$repo" log --reverse --format=%H -S GoGoProtoPackageIsVersion15 HEAD -- proto/lib.go | sed -n '1p')
revision=HEAD
if test -n "$introduction"; then revision="$introduction^"; fi
mkdir -p "$tmp/old" "$tmp/negative"
(cd "$repo" && git archive "$revision") | tar -xf - -C "$tmp/old"
cp "$tmp/map/gogo-standard/pb/"*.go "$tmp/negative/"
cp "$repo/go.sum" "$tmp/negative/go.sum"
cat > "$tmp/negative/go.mod" <<MOD
module example.com/closed-enum-runtime-guard
go 1.15
require github.com/gogo/protobuf v1.3.2
replace github.com/gogo/protobuf => $tmp/old
MOD
if (cd "$tmp/negative" && go build ./...) > "$tmp/error" 2>&1; then
 echo 'closed-enum code compiled against a pre-policy runtime' >&2; exit 1
fi
grep -q GoGoProtoPackageIsVersion15 "$tmp/error"
