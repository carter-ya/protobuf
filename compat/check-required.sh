#!/bin/sh
set -eu
repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT HUP INT TERM
python3 "$repo/compat/required/prepare.py" "$tmp/consumer"
# Runtime immediately before B02 already supports Editions but has the 64-bit
# validation limit. Wide generated types must refuse that runtime explicitly.
mkdir -p "$tmp/old" "$tmp/negative"
# Locate the introduction by content so Stack rebases do not invalidate a
# hard-coded parent SHA. This always selects the runtime immediately before B02.
introduction=$(git -C "$repo" log --reverse --format=%H -S GoGoProtoPackageIsVersion7 HEAD -- proto/lib.go | sed -n '1p')
test -n "$introduction"
(cd "$repo" && git archive "$introduction^") | tar -xf - -C "$tmp/old"
cp "$tmp/consumer/proto2129/proto2129.pb.go" "$tmp/negative/"
cp "$repo/go.sum" "$tmp/negative/go.sum"
cat > "$tmp/negative/go.mod" <<MOD
module example.com/required-runtime-guard
go 1.15
require github.com/gogo/protobuf v1.3.2
replace github.com/gogo/protobuf => $tmp/old
MOD
if (cd "$tmp/negative" && go build ./...) > "$tmp/error" 2>&1; then
 echo 'wide required fields compiled against the incomplete runtime' >&2; exit 1
fi
grep -q GoGoProtoPackageIsVersion7 "$tmp/error"
