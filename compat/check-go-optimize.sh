#!/bin/sh
set -eu
repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
case "$(protoc --version)" in
 'libprotoc 36.2'|'libprotoc 37.0-rc1') ;;
 *) echo 'Go optimization features require a qualified Editions compiler'; exit 0 ;;
esac
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT HUP INT TERM
python3 "$repo/compat/go-optimize/prepare.py" "$tmp/consumer"

introduction=$(git -C "$repo" log --reverse --format=%H -S GoGoProtoPackageIsVersion16 HEAD -- proto/lib.go | sed -n '1p')
revision=HEAD
if test -n "$introduction"; then revision="$introduction^"; fi
mkdir -p "$tmp/old" "$tmp/negative"
(cd "$repo" && git archive "$revision") | tar -xf - -C "$tmp/old"
cp "$tmp/consumer/gogo-2024-CODE_SIZE/pb/"*.go "$tmp/negative/"
cp "$repo/go.sum" "$tmp/negative/go.sum"
cat > "$tmp/negative/go.mod" <<MOD
module example.com/compact-runtime-guard
go 1.15
require github.com/gogo/protobuf v1.3.2
replace github.com/gogo/protobuf => $tmp/old
MOD
if (cd "$tmp/negative" && go build ./...) > "$tmp/error" 2>&1; then
 echo 'compact codec linked a runtime without its shortcut policy' >&2; exit 1
fi
grep -q GoGoProtoPackageIsVersion16 "$tmp/error"
