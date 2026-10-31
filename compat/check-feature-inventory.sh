#!/bin/sh
set -eu
repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
case "$(protoc --version)" in
 'libprotoc 36.2'|'libprotoc 37.0-rc1') ;;
 *) echo 'Feature inventory requires pinned protoc 36.2 or 37.0-rc1' >&2; exit 1;;
esac
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT HUP INT TERM
# No repository include path: descriptors must come from the official compiler.
protoc --include_imports --descriptor_set_out="$tmp/official.pb" google/protobuf/descriptor.proto google/protobuf/go_features.proto
cd "$repo"
go build -o "$tmp/audit" ./compat/feature-audit
"$tmp/audit" "$tmp/official.pb" compat/feature-audit/inventory.json "$repo" "$tmp/report.json"
if test -n "${FEATURE_AUDIT_REPORT:-}"; then cp "$tmp/report.json" "$FEATURE_AUDIT_REPORT"; fi

python3 - "$tmp" "$repo" <<'PYTEST'
import copy,json,pathlib,subprocess,sys
out,repo=map(pathlib.Path,sys.argv[1:]);original=json.loads((repo/'compat/feature-audit/inventory.json').read_text())
for name in ['field','value']:
 bad=copy.deepcopy(original)
 if name=='field':bad['core'].pop()
 else:bad['go'][1]['values']['FUTURE_API']=4
 path=out/(name+'.json');path.write_text(json.dumps(bad))
 result=subprocess.run([out/'audit',out/'official.pb',path,repo,out/'bad-report.json'],stdout=subprocess.PIPE,stderr=subprocess.STDOUT,text=True)
 if result.returncode==0 or 'inventory changed' not in result.stdout:raise RuntimeError('inventory evolution guard failed: '+result.stdout)
print('Feature field/value evolution guards passed')
PYTEST
python3 compat/feature-audit/frontend.py "$tmp/frontend"
