#!/usr/bin/env python3
"""Exercise legacy enum JSON method generation with isolated consumers."""
import argparse
import json
import pathlib
import shutil
import subprocess

ROOT = pathlib.Path(__file__).resolve().parents[2]
HERE = pathlib.Path(__file__).resolve().parent
BACKENDS = ['gogo', 'gogofast', 'gogofaster', 'gogoslick', 'gogotypes']


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('output', type=pathlib.Path)
    parser.add_argument('--official-generator', type=pathlib.Path)
    parser.add_argument('--official-module', default='v1.36.12')
    args = parser.parse_args()
    compiler = subprocess.check_output(['protoc', '--version'], text=True).strip()
    if compiler not in ['libprotoc 36.2', 'libprotoc 37.0-rc1']:
        parser.error('requires the qualified stable or preview compiler')
    out = args.output.resolve()
    out.mkdir(parents=True)
    checks = []
    backends = ['official'] if args.official_generator else BACKENDS
    editions = ['2023', '2024'] + (['2026'] if args.official_module != 'v1.36.12' or not args.official_generator else [])
    for backend in backends:
        driver = args.official_generator.resolve() if args.official_generator else out / backend
        if not args.official_generator:
            subprocess.run(['go', 'build', '-o', driver, './protoc-gen-' + backend], cwd=ROOT, check=True)
        for edition in editions:
            for value in ['default', 'true', 'false']:
                dest = out / (backend + '-' + edition + '-' + value)
                (dest / 'pb').mkdir(parents=True)
                source = (HERE / 'enums.proto').read_text().replace('"2023"', '"' + edition + '"')
                source = source.replace('// FILE_LEGACY_JSON', '' if value == 'default' else 'option features.(pb.go).legacy_unmarshal_json_enum = ' + value + ';')
                if edition == '2023':
                    source = source.replace('import option', 'import')
                (dest / 'enums.proto').write_text(source)
                subprocess.run(['protoc', '-I' + str(dest), '--plugin=protoc-gen-test=' + str(driver),
                    '--test_out=paths=source_relative:' + str(dest / 'pb'), 'enums.proto'], check=True)
                code = (HERE / 'consumer_test.go.txt').read_text().replace('INHERITED', 'true' if value == 'true' else 'false')
                (dest / 'consumer_test.go').write_text(code)
                if args.official_generator:
                    module = ('go 1.23\n' if args.official_module == 'v1.36.12' else 'go 1.26\n') + 'require google.golang.org/protobuf ' + args.official_module + '\n'
                    sums = (ROOT / 'compat/qualify/go.sum').read_text() + (ROOT / 'compat/preview/go.sum').read_text()
                else:
                    module = 'go 1.15\nrequire github.com/gogo/protobuf v1.3.2\nreplace github.com/gogo/protobuf => ' + str(ROOT) + '\n'
                    sums = (ROOT / 'go.sum').read_text()
                (dest / 'go.mod').write_text('module example.com/go-enum-json\n' + module)
                (dest / 'go.sum').write_text(sums)
                for mode, flags in [('default', []), ('purego', ['-tags=purego'])]:
                    subprocess.run(['go', 'test', '-count=1'] + flags + ['./...'], cwd=dest, check=True)
                    checks.append({'backend': backend, 'edition': edition, 'file_value': value, 'mode': mode})
    (out / 'qualification.json').write_text(json.dumps({'compiler': compiler, 'checks': checks, 'status': 'passed'}, indent=2) + '\n')
    print('Enum legacy JSON method inheritance and behavior passed:', len(checks), 'consumer runs')


if __name__ == '__main__':
    main()
