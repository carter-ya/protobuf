#!/bin/sh
set -eu
repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT HUP INT TERM
python3 "$repo/compat/oneof-merge/prepare.py" "$tmp/consumers"
python3 "$repo/compat/oneof-merge/negative.py" "$tmp/consumers"
mkdir -p "$tmp/old"
introduction=$(git -C "$repo" log --reverse --format=%H -S GoGoProtoPackageIsVersion12 HEAD -- proto/lib.go | sed -n '1p')
revision=HEAD
if test -n "$introduction"; then revision="$introduction^"; fi
(cd "$repo" && git archive "$revision") | tar -xf - -C "$tmp/old"
consumer="$tmp/consumers/proto3-standard-gogo"
cat > "$consumer/go.mod" <<MOD
module example.com/oneof-merge
go 1.15
require github.com/gogo/protobuf v1.3.2
replace github.com/gogo/protobuf => $tmp/old
MOD
if (cd "$consumer" && go build ./...) > "$tmp/error" 2>&1; then
 echo 'oneof merge code compiled against a runtime without its metadata' >&2; exit 1
fi
grep -q GoGoProtoPackageIsVersion12 "$tmp/error"
