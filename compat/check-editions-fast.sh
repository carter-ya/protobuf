#!/bin/sh
set -eu
repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
case "$(protoc --version)" in 'libprotoc 36.2') ;; *) echo 'Editions fast matrix runs with pinned protoc 36.2'; exit 0;; esac
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT HUP INT TERM
python3 "$repo/compat/fast/prepare.py" "$tmp/consumer"
# Complex bindings require B05's group/extension behavior, even on a runtime
# that already understands Editions scalar fast codecs.
introduction=$(git -C "$repo" log --reverse --format=%H -S GoGoProtoPackageIsVersion9 HEAD -- proto/lib.go | sed -n '1p')
revision=HEAD
if test -n "$introduction"; then revision="$introduction^"; fi
mkdir -p "$tmp/pre-complex" "$tmp/negative-complex"
(cd "$repo" && git archive "$revision") | tar -xf - -C "$tmp/pre-complex"
cp "$tmp/consumer/complex-2023-gogofast/pb/"*.go "$tmp/negative-complex/"
cp "$repo/go.sum" "$tmp/negative-complex/"
cat > "$tmp/negative-complex/go.mod" <<MOD
module example.com/complex-runtime-guard
go 1.15
require github.com/gogo/protobuf v1.3.2
replace github.com/gogo/protobuf => $tmp/pre-complex
MOD
if (cd "$tmp/negative-complex" && go build ./...) > "$tmp/error" 2>&1; then
 echo 'complex Editions code compiled against the incomplete runtime' >&2; exit 1
fi
grep -q GoGoProtoPackageIsVersion9 "$tmp/error"
