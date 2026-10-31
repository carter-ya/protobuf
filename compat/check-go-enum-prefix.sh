#!/bin/sh
set -eu
repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
case "$(protoc --version)" in
 'libprotoc 36.2'|'libprotoc 37.0-rc1') ;;
 *) echo 'Go enum prefix features require a qualified Editions compiler'; exit 0 ;;
esac
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT HUP INT TERM
python3 "$repo/compat/go-enum-prefix/prepare.py" "$tmp/consumer"
