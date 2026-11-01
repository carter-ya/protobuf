#!/usr/bin/env python3
"""Qualify native reflection, HYBRID APIs and scoped Editions settings."""
import argparse
import json
import pathlib
import shutil
import subprocess

ROOT = pathlib.Path(__file__).resolve().parents[2]
HERE = pathlib.Path(__file__).resolve().parent
BACKENDS = ['gogo', 'gogofast', 'gogofaster', 'gogoslick', 'gogotypes']
DEFAULT_RUNTIME = 'v1.36.13-0.20261007182434-0c373ff94f29'


def run(command, cwd=ROOT):
    subprocess.run([str(x) for x in command], cwd=cwd, check=True)


def module(dest, version):
    (dest / 'go.mod').write_text('module example.com/native\n\ngo 1.23\nrequire (\n'
        ' github.com/gogo/protobuf v1.3.2\n github.com/gogo/protobuf/reflect v0.0.0\n'
        ' google.golang.org/protobuf ' + version + '\n)\n'
        'replace github.com/gogo/protobuf => ' + str(ROOT) + '\n'
        'replace github.com/gogo/protobuf/reflect => ' + str(ROOT / 'reflect') + '\n')
    (dest / 'go.sum').write_text((ROOT / 'reflect/go.sum').read_text() + (ROOT / 'compat/preview/go.sum').read_text())


def diagnostics(out, driver):
    dest = out / 'diagnostics'
    dest.mkdir()
    cases = [
        ('bad-bool', 'proto_reflect=yes', '', '', 'proto_reflect must be'),
        ('opaque', 'default_api_level=API_OPAQUE', '', '', 'OPAQUE is not implemented'),
        ('closed-policy', 'proto_reflect=true,enforce_closed_enums=false', '', '', 'conflicts with native reflection'),
        ('merge-policy', 'proto_reflect=true,merge_required_fields=false', '', '', 'conflicts with native reflection'),
        ('unknown', 'proto_reflect=true', 'option (gogoproto.goproto_unrecognized) = false;', '', 'requires unknown-field storage'),
        ('getters', 'default_api_level=API_HYBRID', 'option (gogoproto.goproto_getters) = false;', '', 'HYBRID requires getters'),
        ('value', 'proto_reflect=true', '', 'Test child = 2 [(gogoproto.nullable) = false];', 'requires independent native storage'),
        ('custom', 'proto_reflect=true', '', 'bytes custom = 2 [(gogoproto.customtype) = "time.Duration"];', 'custom/std/embedded mapping is unsupported'),
        ('conflict', 'default_api_level=API_HYBRID', '', 'int32 set_value = 2;', 'conflicts with existing Go API'),
    ]
    for name, params, options, extra, expected in cases:
        source = ('syntax = "proto2"; package diagnostic; import "gogoproto/gogo.proto"; '
                  'option go_package = "example.com/diagnostic"; message Test {' + options +
                  'optional int32 value = 1; ' + extra.replace('Test child', 'optional Test child').replace('bytes custom', 'optional bytes custom').replace('int32 set_value', 'optional int32 set_value') + '}')
        (dest / 'bad.proto').write_text(source)
        result = subprocess.run(['protoc', '-I' + str(dest), '-I' + str(ROOT), '-I' + str(ROOT / 'protobuf'),
            '--plugin=protoc-gen-test=' + str(driver), '--test_out=' + params + ':' + str(dest), 'bad.proto'],
            cwd=ROOT, text=True, stdout=subprocess.PIPE, stderr=subprocess.STDOUT)
        if result.returncode == 0 or expected not in result.stdout:
            raise RuntimeError(name + ' expected diagnostic: ' + expected + '\n' + result.stdout)
    print('Native/HYBRID diagnostic regressions:', len(cases))


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('output', type=pathlib.Path)
    parser.add_argument('--backend', action='append')
    parser.add_argument('--flavor', action='append')
    parser.add_argument('--official-generator', type=pathlib.Path)
    parser.add_argument('--runtime-version', default=DEFAULT_RUNTIME)
    args = parser.parse_args()
    out = args.output.resolve()
    out.mkdir(parents=True)
    compiler = subprocess.check_output(['protoc', '--version'], text=True).strip()
    checks = []
    oracle = None
    if args.official_generator:
        reference = out / 'official'
        reference.mkdir()
        run(['protoc', '-I.', '--plugin=protoc-gen-go=' + str(args.official_generator.resolve()),
             '--go_out=paths=source_relative,default_api_level=API_HYBRID:' + str(reference),
             'reflect/internal/testpb/schema.proto'])
        oracle = out / 'compare-api'
        run(['go', 'build', '-o', oracle, './compat/hybrid/compare-api'])
    for backend in args.backend or BACKENDS:
        driver = out / backend
        run(['go', 'build', '-o', driver, './protoc-gen-' + backend])
        if backend == (args.backend or BACKENDS)[0]:
            diagnostics(out, driver)
        for flavor in args.flavor or ['proto3', 'proto2', '2023', '2024', '2026']:
            modes = [('source', '')] if flavor.isdigit() else [('reflect', 'proto_reflect=true'), ('hybrid', 'default_api_level=API_HYBRID')]
            for mode, parameters in modes:
                dest = out / (backend + '-' + flavor + '-' + mode)
                (dest / 'pb').mkdir(parents=True)
                if flavor.isdigit():
                    source = (HERE / 'edition.proto').read_text().replace('"2024"', '"' + flavor + '"')
                    if flavor == '2023':
                        source = source.replace('import option ', 'import ').replace('option features.enforce_naming_style = STYLE_LEGACY;\n', '')
                    if flavor == '2026':
                        source = source.replace('// JSON_IMPORT', 'import option "google/protobuf/json_enumvalue_options.proto";')
                        source = source.replace('ONE = 1;  // JSON_VALUE', 'ONE = 1 [(pb.enumvalue.json).string = "named-one"];')
                    (dest / 'edition.proto').write_text(source)
                    run(['protoc', '-I' + str(dest), '--plugin=protoc-gen-test=' + str(driver),
                        '--test_out=paths=source_relative:' + str(dest / 'pb'), 'edition.proto'])
                    (dest / 'consumer_test.go').write_text((HERE / 'edition_test.go.txt').read_text().replace('EDITION', flavor))
                else:
                    inputs = ['reflect/internal/testpb/schema.proto']
                    if flavor == 'proto2':
                        inputs.append('reflect/internal/testpb/closed.proto')
                    run(['protoc', '-I.', '-Iprotobuf', '--plugin=protoc-gen-test=' + str(driver),
                        '--test_out=paths=source_relative,' + parameters + ':' + str(dest / 'pb')] + inputs)
                    for src in (dest / 'pb/reflect/internal/testpb').glob('*.go'):
                        shutil.move(src, dest / 'pb' / src.name)
                    shutil.rmtree(dest / 'pb/reflect')
                    (dest / 'consumer_test.go').write_text((HERE / 'consumer_test.go.txt').read_text().replace('HYBRID', str(mode == 'hybrid').lower()))
                    if mode == 'hybrid':
                        shutil.copyfile(HERE / 'hybrid_test.go.txt', dest / 'hybrid_test.go')
                    if flavor == 'proto2':
                        shutil.copyfile(HERE / 'closed_test.go.txt', dest / 'closed_test.go')
                if oracle and flavor == 'proto3' and mode == 'hybrid':
                    run([oracle, reference / 'reflect/internal/testpb/schema.pb.go', dest / 'pb/schema.pb.go'])
                module(dest, args.runtime_version)
                for variant, flags in [('default', []), ('purego', ['-tags=purego'])]:
                    run(['go', 'test', '-mod=mod', '-count=1'] + flags + ['./...'], dest)
                    checks.append({'backend': backend, 'flavor': flavor, 'mode': mode, 'variant': variant})
    (out / 'qualification.json').write_text(json.dumps({'compiler': compiler, 'runtime': args.runtime_version,
        'checks': checks, 'status': 'passed'}, indent=2) + '\n')
    print('Native/HYBRID consumer runs:', len(checks))


if __name__ == '__main__':
    main()
