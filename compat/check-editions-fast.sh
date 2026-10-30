#!/bin/sh
set -eu
repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
case "$(protoc --version)" in 'libprotoc 36.2') ;; *) echo 'Editions fast matrix runs with pinned protoc 36.2'; exit 0;; esac
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT HUP INT TERM
python3 "$repo/compat/fast/prepare.py" "$tmp/consumer"
# New generated codecs must refuse the Editions-capable pre-fast runtime.
introduction=$(git -C "$repo" log --reverse --format=%H -S GoGoProtoPackageIsVersion8 HEAD -- proto/lib.go | sed -n '1p')
revision=HEAD
if test -n "$introduction"; then revision="$introduction^"; fi
mkdir -p "$tmp/old" "$tmp/negative"
(cd "$repo" && git archive "$revision") | tar -xf - -C "$tmp/old"
cp "$tmp/consumer/2023-gogofast/pb/scalars.pb.go" "$tmp/negative/"
cp "$repo/go.sum" "$tmp/negative/"
cat > "$tmp/negative/go.mod" <<MOD
module example.com/fast-runtime-guard
go 1.15
require github.com/gogo/protobuf v1.3.2
replace github.com/gogo/protobuf => $tmp/old
MOD
if (cd "$tmp/negative" && go build ./...) > "$tmp/error" 2>&1; then
 echo 'fast Editions code compiled against the incomplete runtime' >&2; exit 1
fi
grep -q GoGoProtoPackageIsVersion8 "$tmp/error"
