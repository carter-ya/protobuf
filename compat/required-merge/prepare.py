#!/usr/bin/env python3
"""Qualify final required initialization and preserve unmarked legacy policy."""
import argparse
import pathlib
import re
import shutil
import subprocess

ROOT = pathlib.Path(__file__).resolve().parents[2]
BACKENDS = ['gogo', 'gogofast', 'gogofaster', 'gogoslick', 'gogotypes']


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('output', type=pathlib.Path)
    args = parser.parse_args()
    version = subprocess.check_output(['protoc', '--version'], text=True).strip()
    if version not in ['libprotoc 3.21.8', 'libprotoc 36.2', 'libprotoc 37.0-rc1']:
        parser.error('use a pinned qualified compiler')
    syntaxes = ['proto2', 'proto3'] + ([] if version == 'libprotoc 3.21.8' else ['2023', '2024', '2026'])
    output = args.output.resolve()
    output.mkdir(parents=True)
    for backend in BACKENDS:
        driver = output / backend
        subprocess.run(['go', 'build', '-o', str(driver), './protoc-gen-' + backend], cwd=ROOT, check=True)
        for syntax in syntaxes:
            for mode in (['legacy', 'standard'] if syntax.startswith('proto') else ['standard']):
                dest = output / (syntax + '-' + mode + '-' + backend)
                (dest / 'pb').mkdir(parents=True)
                leaf = (ROOT / 'compat/required-merge/leaf.proto').read_text()
                record = (ROOT / 'compat/required-merge/record.proto').read_text()
                if syntax.startswith('proto'):
                    leaf = leaf.replace('edition = "2023";', 'syntax = "proto2";')
                    leaf = leaf.replace('int32 a = 1 [features.field_presence = LEGACY_REQUIRED]', 'required int32 a = 1 [default = 0]').replace(
                        'int32 b = 2 [features.field_presence = LEGACY_REQUIRED]', 'required int32 b = 2 [default = 0]').replace(
                        'bytes data = 3 [features.field_presence = LEGACY_REQUIRED]', 'required bytes data = 3').replace('  string text', '  optional string text')
                    record = record.replace('edition = "2023";', 'syntax = "' + syntax + '";').replace(' [features.message_encoding = DELIMITED]', '')
                    if syntax == 'proto2':
                        record = re.sub(r'^  (Leaf|string) ', r'  optional \1 ', record, flags=re.MULTILINE)
                else:
                    leaf = leaf.replace('"2023"', '"' + syntax + '"')
                    record = record.replace('"2023"', '"' + syntax + '"')
                    if syntax == '2026':
                        leaf = leaf.replace('message Leaf {', 'export message Leaf {')
                (dest / 'leaf.proto').write_text(leaf)
                (dest / 'record.proto').write_text(record)
                params = ['paths=source_relative'] + (['merge_required_fields=true', 'merge_oneof_messages=true'] if mode == 'standard' else [])
                subprocess.run(['protoc', '-I' + str(dest), '--plugin=protoc-gen-test=' + str(driver),
                    '--test_out=' + ','.join(params) + ':' + str(dest / 'pb'), 'leaf.proto', 'record.proto'], check=True)
                if mode == 'standard' and 'GoGoProtoPackageIsVersion14' not in (dest / 'pb/record.pb.go').read_text():
                    raise RuntimeError('required merge runtime guard missing')
                code = (ROOT / 'compat/required-merge/consumer_test.go.txt').read_text().replace('STANDARD', str(mode == 'standard').lower()).replace(
                    'EDITIONS', str(not syntax.startswith('proto')).lower()).replace('CHECKED_UTF8', str(syntax != 'proto2').lower())
                (dest / 'consumer_test.go').write_text(code)
                (dest / 'go.mod').write_text('module example.com/gogo-required-merge\n\ngo 1.15\nrequire github.com/gogo/protobuf v1.3.2\nreplace github.com/gogo/protobuf => ' + str(ROOT) + '\n')
                shutil.copyfile(ROOT / 'go.sum', dest / 'go.sum')
                subprocess.run(['go', 'test', './...'], cwd=dest, check=True)
                subprocess.run(['go', 'test', '-tags=purego', './...'], cwd=dest, check=True)
    negative = output / 'negative-inputs'
    negative.mkdir()
    (negative / 'bad.proto').write_text('syntax="proto2";package bad;import "gogoproto/gogo.proto";message Bad {required int32 value=1 [(gogoproto.nullable)=false];}\n')
    for backend in BACKENDS:
        result = subprocess.run(['protoc', '-I' + str(negative), '-I' + str(ROOT), '-I' + str(ROOT / 'protobuf'),
            '--plugin=protoc-gen-test=' + str(output / backend), '--test_out=merge_required_fields=true:' + str(negative), 'bad.proto'],
            stdout=subprocess.PIPE, stderr=subprocess.STDOUT, text=True)
        if result.returncode == 0 or 'independent required presence' not in result.stdout:
            raise RuntimeError('unrepresentable required presence was not diagnosed: ' + result.stdout)
    (negative / 'child.proto').write_text('syntax="proto2";package bad;import "gogoproto/gogo.proto";message Leaf {required int32 value=1 [default=0];}message Root {optional Leaf child=1 [(gogoproto.nullable)=false];}\n')
    for backend in BACKENDS:
        result = subprocess.run(['protoc', '-I' + str(negative), '-I' + str(ROOT), '-I' + str(ROOT / 'protobuf'),
            '--plugin=protoc-gen-test=' + str(output / backend), '--test_out=merge_required_fields=true:' + str(negative), 'child.proto'],
            stdout=subprocess.PIPE, stderr=subprocess.STDOUT, text=True)
        if result.returncode == 0 or 'independent message presence' not in result.stdout:
            raise RuntimeError('flattened optional required child was not diagnosed: ' + result.stdout)
    alternate = output / 'gofast'
    subprocess.run(['go', 'build', '-o', str(alternate), './protoc-gen-gofast'], cwd=ROOT, check=True)
    (negative / 'plain.proto').write_text('syntax="proto2";package bad;message Bad {required int32 value=1;}\n')
    result = subprocess.run(['protoc', '-I' + str(negative), '--plugin=protoc-gen-test=' + str(alternate),
        '--test_out=merge_required_fields=true:' + str(negative), 'plain.proto'], stdout=subprocess.PIPE, stderr=subprocess.STDOUT, text=True)
    if result.returncode == 0 or 'requires the gogo runtime' not in result.stdout:
        raise RuntimeError('alternate runtime was not diagnosed: ' + result.stdout)
    result = subprocess.run(['protoc', '-I' + str(negative), '--plugin=protoc-gen-test=' + str(output / 'gogo'),
        '--test_out=merge_required_fields=invalid:' + str(negative), 'plain.proto'], stdout=subprocess.PIPE, stderr=subprocess.STDOUT, text=True)
    if result.returncode == 0 or 'must be true or false' not in result.stdout:
        raise RuntimeError('invalid required merge parameter accepted: ' + result.stdout)
    print('Required merge qualification passed')


if __name__ == '__main__':
    main()
