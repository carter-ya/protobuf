#!/bin/sh
set -eu
repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
case "$(protoc --version)" in 'libprotoc 36.2') ;; *) echo 'Editions plugin matrix runs with pinned protoc 36.2'; exit 0;; esac
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT HUP INT TERM
python3 "$repo/compat/plugins/prepare.py" "$tmp/consumer"
