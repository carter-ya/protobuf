#!/usr/bin/env python3
"""Exercise private OPAQUE generation through all maintained backends."""
import argparse
import json
import pathlib
import shutil
import subprocess
import sys

ROOT = pathlib.Path(__file__).resolve().parents[2]
HERE = pathlib.Path(__file__).resolve().parent
sys.path.insert(0, str(ROOT / 'compat/hybrid'))
from prepare import BACKENDS, module, run


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('output', type=pathlib.Path)
    parser.add_argument('--backend', action='append', choices=BACKENDS)
    parser.add_argument('--flavor', action='append', choices=['proto3', 'proto2', '2023', '2024', '2026'])
    parser.add_argument('--official-generator', type=pathlib.Path)
    args = parser.parse_args()
    out = args.output.resolve()
    out.mkdir(parents=True)
    checks = []
    oracle = None
    if args.official_generator:
        reference = out / 'official'
        reference.mkdir()
        run(['protoc', '-I.', '--plugin=protoc-gen-go=' + str(args.official_generator.resolve()),
             '--go_out=paths=source_relative,default_api_level=API_OPAQUE:' + str(reference),
             'reflect/internal/testpb/schema.proto'])
        oracle = out / 'compare-api'
        run(['go', 'build', '-o', oracle, './compat/hybrid/compare-api'])
    for backend in args.backend or BACKENDS:
        driver = out / backend
        run(['go', 'build', '-o', driver, './protoc-gen-' + backend])
        names = out / (backend + '-names')
        (names / 'pb').mkdir(parents=True)
        run(['protoc', '-I' + str(HERE), '--plugin=protoc-gen-test=' + str(driver),
             '--test_out=paths=source_relative,default_api_level=API_OPAQUE:' + str(names / 'pb'), 'names.proto', 'defaults.proto'])
        shutil.copyfile(HERE / 'names_test.go.txt', names / 'names_test.go')
        module(names, 'v1.36.13-0.20261007182434-0c373ff94f29')
        for variant, flags in [('default', []), ('purego', ['-tags=purego']), ('race', ['-race'])]:
            run(['go', 'test', '-mod=mod', '-count=1'] + flags + ['./...'], names)
            checks.append({'backend': backend, 'flavor': 'names', 'variant': variant})

        for flavor in args.flavor or ['proto3', 'proto2', '2023', '2024', '2026']:
            dest = out / (backend + '-' + flavor)
            (dest / 'pb').mkdir(parents=True)
            if flavor.isdigit():
                source = (ROOT / 'compat/hybrid/edition.proto').read_text().replace('"2024"', '"' + flavor + '"').replace('API_HYBRID', 'API_OPAQUE')
                if flavor == '2023':
                    source = source.replace('import option ', 'import ').replace('option features.enforce_naming_style = STYLE_LEGACY;\n', '')
                if flavor == '2026':
                    source = source.replace('// JSON_IMPORT', 'import option "google/protobuf/json_enumvalue_options.proto";').replace('ONE = 1;  // JSON_VALUE', 'ONE = 1 [(pb.enumvalue.json).string = "named-one"];')
                (dest / 'edition.proto').write_text(source)
                run(['protoc', '-I' + str(dest), '--plugin=protoc-gen-test=' + str(driver), '--test_out=paths=source_relative:' + str(dest / 'pb'), 'edition.proto'])
                test = (ROOT / 'compat/hybrid/edition_test.go.txt').read_text().replace('EDITION', flavor)
                test = test.replace('&pb.Child{Name:gogo.String("child")}', 'pb.Child_builder{Name:gogo.String("child")}.Build()').replace('map[string]*pb.Child{"x":{Name:gogo.String("mapped")}}', 'map[string]*pb.Child{"x":pb.Child_builder{Name:gogo.String("mapped")}.Build()}')
                (dest / 'consumer_test.go').write_text(test)
            else:
                inputs = ['reflect/internal/testpb/schema.proto']
                if flavor == 'proto2':
                    inputs.append('reflect/internal/testpb/closed.proto')
                run(['protoc', '-I.', '-Iprotobuf', '--plugin=protoc-gen-test=' + str(driver), '--test_out=paths=source_relative,default_api_level=API_OPAQUE:' + str(dest / 'pb')] + inputs)
                for src in (dest / 'pb/reflect/internal/testpb').glob('*.go'):
                    shutil.move(src, dest / 'pb' / src.name)
                shutil.rmtree(dest / 'pb/reflect')
                shutil.copyfile(HERE / 'consumer_test.go.txt', dest / 'consumer_test.go')
                if flavor == 'proto2':
                    test = (ROOT / 'compat/hybrid/closed_test.go.txt').read_text().replace('n.Status!=nil', 'n.HasStatus()').replace('n.Child.GetValue()', 'n.GetChild().GetValue()')
                    (dest / 'closed_test.go').write_text(test)
            module(dest, 'v1.36.13-0.20261007182434-0c373ff94f29')
            if flavor == 'proto3':
                if oracle:
                    run([oracle, reference / 'reflect/internal/testpb/schema.pb.go', dest / 'pb/schema.pb.go', '--opaque'])
                negative = dest / 'direct_fields.go'
                negative.write_text('package consumer\nimport pb "example.com/native/pb"\nvar _ = (&pb.Node{}).Count\n')
                try:
                    result = subprocess.run(['go', 'test', '-mod=mod', './...'], cwd=dest, text=True, stdout=subprocess.PIPE, stderr=subprocess.STDOUT)
                    if result.returncode == 0 or 'has no field or method Count' not in result.stdout:
                        raise RuntimeError('OPAQUE direct-field compile guard failed:\n' + result.stdout)
                finally:
                    negative.unlink()

            for variant, flags in [('default', []), ('purego', ['-tags=purego']), ('race', ['-race'])]:
                run(['go', 'test', '-mod=mod', '-count=1'] + flags + ['./...'], dest)
                checks.append({'backend': backend, 'flavor': flavor, 'variant': variant})
    (out / 'qualification.json').write_text(json.dumps({'status': 'passed', 'checks': checks}, indent=2) + '\n')


if __name__ == '__main__':
    main()
