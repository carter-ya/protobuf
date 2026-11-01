#!/bin/sh
set -eu
repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
cd "$repo"
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT HUP INT TERM
python3 compat/hybrid/prepare.py "$tmp/consumers" "$@"
python3 compat/hybrid/mixed.py "$tmp/mixed"
