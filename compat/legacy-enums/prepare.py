#!/usr/bin/env python3
"""Qualify opt-in definition-scoped enum policies and unmodified legacy controls."""
import argparse
import json
import os
import pathlib
import re
import shutil
import subprocess

ROOT = pathlib.Path(__file__).resolve().parents[2]
HERE = pathlib.Path(__file__).resolve().parent
BACKENDS = ['gogo', 'gogofast', 'gogofaster', 'gogoslick', 'gogotypes']


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('output', type=pathlib.Path)
    parser.add_argument('--byte-extensions', action='store_true')
    args = parser.parse_args()
    out = args.output.resolve()
    out.mkdir(parents=True)
    version = subprocess.check_output(['protoc', '--version'], text=True).strip()
    if version not in ['libprotoc 3.21.8', 'libprotoc 36.2', 'libprotoc 37.0-rc1']:
        parser.error('use a pinned compiler')
    edition = version != 'libprotoc 3.21.8'
    checks = []
    for backend in BACKENDS:
        driver = out / backend
        subprocess.run(['go', 'build', '-o', driver, './protoc-gen-' + backend], cwd=ROOT, check=True)
        for standard in [False, True]:
            dest = out / (backend + ('-standard' if standard else '-legacy'))
            (dest / 'pb').mkdir(parents=True)
            names = ['open.proto', 'record.proto'] + (['edition.proto'] if edition else [])
            for name in names:
                source = (HERE / name).read_text()
                if not edition:
                    source = source.replace('import "edition.proto";', '').replace('  optional EditionClosed imported_closed = 7;', '')
                # Legacy controls retain pointer fields without adopting the new policy.
                if not standard and name == 'record.proto':
                    source = source.replace('option go_package = "example.com/legacy-enums/pb";',
                        'option go_package = "example.com/legacy-enums/pb";\noption (gogoproto.goproto_unrecognized_all) = true;').replace(
                        'optional Proto2Closed status = 1;', 'optional Proto2Closed status = 1 [(gogoproto.nullable) = true];').replace(
                        'required Proto2Closed status = 1;', 'required Proto2Closed status = 1 [default = PROTO2_CLOSED_FIRST];').replace(
                        'optional EditionClosed imported_closed = 7;', 'optional EditionClosed imported_closed = 7 [(gogoproto.nullable) = true];').replace(
                        'optional Proto3Open imported_open = 6;', 'optional Proto3Open imported_open = 6 [(gogoproto.nullable) = true];')
                if args.byte_extensions and name == 'record.proto':
                    source = source.replace('option go_package =', 'option (gogoproto.goproto_extensions_map_all) = false;\noption go_package =')
                (dest / name).write_text(source)
            params = ['paths=source_relative'] + (['enforce_closed_enums=true', 'merge_required_fields=true'] if standard else [])
            subprocess.run(['protoc', '-I' + str(dest), '-I' + str(ROOT), '-I' + str(ROOT / 'protobuf'),
                '--plugin=protoc-gen-test=' + str(driver), '--test_out=' + ','.join(params) + ':' + str(dest / 'pb')] + names, check=True)
            code = (HERE / 'consumer_test.go.txt').read_text().replace('STANDARD', str(standard).lower()).replace('EDITION_IMPORT', str(edition).lower())
            if not edition:
                code = re.sub(r'(?ms)^\s*// BEGIN EDITION.*?// END EDITION', '', code)
            (dest / 'consumer_test.go').write_text(code)
            (dest / 'go.mod').write_text('module example.com/legacy-enums\ngo 1.15\nrequire github.com/gogo/protobuf v1.3.2\nreplace github.com/gogo/protobuf => ' + str(ROOT) + '\n')
            shutil.copyfile(ROOT / 'go.sum', dest / 'go.sum')
            for mode, flags in [('default', []), ('purego', ['-tags=purego'])]:
                corpus = dest / mode
                corpus.mkdir()
                subprocess.run(['go', 'test', '-count=1'] + flags + ['./...'], cwd=dest, env=dict(os.environ, GOGO_ENUM_CORPUS=str(corpus)), check=True)
                oracle_count = 0
                if standard:
                    # Compare independent C++ decoding of input and re-encoding.
                    # Exclude proto2->proto3 enum imports: C++ has a documented
                    # definition-scope conformance exception for those imports.
                    for original in sorted(corpus.glob('*.input')):
                        def decode(data):
                            return subprocess.check_output(['protoc', '-I' + str(dest), '-I' + str(ROOT), '-I' + str(ROOT / 'protobuf'),
                                '--decode=qualification.legacyenum.Record', 'record.proto'], input=data)
                        expected = decode(original.read_bytes())
                        actual = decode(original.with_suffix('.output').read_bytes())
                        if actual != expected:
                            raise RuntimeError('C++ semantic cross-read differs for ' + str(original) + ': ' + repr((expected, actual)))
                        original.with_suffix('.cpp.txt').write_bytes(actual)
                        oracle_count += 1
                checks.append({'backend': backend, 'standard': standard, 'mode': mode, 'cpp_cross_reads': oracle_count})
            if standard and 'GoGoProtoPackageIsVersion15' not in (dest / 'pb/record.pb.go').read_text():
                raise RuntimeError('closed-enum guard missing')
    negative = out / 'negative'
    negative.mkdir()
    prefix = 'syntax="proto2";package bad;import "gogoproto/gogo.proto";enum State {FIRST=5;SECOND=6;}'
    cases = [
        ('unknown-storage', prefix + 'option (gogoproto.goproto_unrecognized_all)=false;message Bad {optional State value=1;}', 'unknown-field storage'),
        ('nullable', prefix + 'message Bad {optional State value=1 [(gogoproto.nullable)=false];}', 'independent enum presence'),
        ('alternate-runtime', prefix + 'option (gogoproto.gogoproto_import)=false;message Bad {optional State value=1;}', 'requires the gogo runtime'),
        ('invalid-flag', prefix + 'message Bad {optional State value=1;}', 'must be true or false'),
    ]
    for backend in BACKENDS:
        for name, schema, diagnostic in cases:
            (negative / 'bad.proto').write_text(schema + '\n')
            value = 'invalid' if name == 'invalid-flag' else 'true'
            process = subprocess.run(['protoc', '-I' + str(negative), '-I' + str(ROOT), '-I' + str(ROOT / 'protobuf'),
                '--plugin=protoc-gen-test=' + str(out / backend), '--test_out=enforce_closed_enums=' + value + ':' + str(negative), 'bad.proto'],
                stdout=subprocess.PIPE, stderr=subprocess.STDOUT, text=True)
            (negative / (backend + '-' + name + '.log')).write_text(process.stdout)
            if process.returncode == 0 or diagnostic not in process.stdout:
                raise RuntimeError('unsupported closed-enum policy was not diagnosed: ' + process.stdout)
    report = {'compiler': version, 'source_head': subprocess.check_output(['git', 'rev-parse', 'HEAD'], cwd=ROOT, text=True).strip(),
              'working_tree': subprocess.check_output(['git', 'status', '--porcelain'], cwd=ROOT, text=True),
              'extension_storage': 'bytes' if args.byte_extensions else 'map', 'checks': checks, 'negative_checks': len(BACKENDS) * len(cases), 'status': 'passed'}
    (out / 'qualification.json').write_text(json.dumps(report, indent=2) + '\n')
    print('All five closed-enum backends, legacy controls, C++ cross-reads and diagnostics passed')


if __name__ == '__main__':
    main()
