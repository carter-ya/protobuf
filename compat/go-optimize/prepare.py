#!/usr/bin/env python3
"""Qualify explicit optimization strategies, scope overrides and mixed graphs."""
import argparse
import json
import pathlib
import shutil
import subprocess

ROOT = pathlib.Path(__file__).resolve().parents[2]
HERE = pathlib.Path(__file__).resolve().parent
BACKENDS = ['gogo', 'gogofast', 'gogofaster', 'gogoslick', 'gogotypes']
MODES = ['default', 'OPTIMIZE_MODE_UNSPECIFIED', 'SPEED', 'CODE_SIZE']


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('output', type=pathlib.Path)
    args = parser.parse_args()
    compiler = subprocess.check_output(['protoc', '--version'], text=True).strip()
    if compiler not in ['libprotoc 36.2', 'libprotoc 37.0-rc1']:
        parser.error('use a qualified Editions compiler')
    out = args.output.resolve()
    out.mkdir(parents=True)
    checks, sizes = [], []
    for backend in BACKENDS:
        driver = out / backend
        subprocess.run(['go', 'build', '-o', driver, './protoc-gen-' + backend], cwd=ROOT, check=True)
        for edition in ['2024', '2026']:
            file_sizes = {}
            for mode in MODES:
                dest = out / (backend + '-' + edition + '-' + mode)
                (dest / 'pb').mkdir(parents=True)
                source = (HERE / 'schema.proto').read_text().replace('"2024"', '"' + edition + '"')
                source = source.replace('// FILE_OPTIMIZE', '' if mode == 'default' else 'option features.(pb.go).optimize_mode = ' + mode + ';')
                (dest / 'schema.proto').write_text(source)
                subprocess.run(['protoc', '-I' + str(dest), '--plugin=protoc-gen-test=' + str(driver),
                    '--test_out=paths=source_relative:' + str(dest / 'pb'), 'schema.proto'], check=True)
                generated = (dest / 'pb/schema.pb.go').read_text()
                if 'GoGoProtoPackageIsVersion16' not in generated:
                    raise RuntimeError('compact codec runtime guard missing')
                file_sizes[mode] = len(generated.encode())
                code = (HERE / 'consumer_test.go.txt').read_text().replace('FILE_MODE', str(1 if mode == 'SPEED' else 2 if mode == 'CODE_SIZE' else 0)).replace('BACKEND_FAST', str(backend != 'gogo').lower())
                (dest / 'consumer_test.go').write_text(code)
                (dest / 'go.mod').write_text('module example.com/optimize\ngo 1.15\nrequire github.com/gogo/protobuf v1.3.2\nreplace github.com/gogo/protobuf => ' + str(ROOT) + '\n')
                shutil.copyfile(ROOT / 'go.sum', dest / 'go.sum')
                for variant, flags in [('default', []), ('purego', ['-tags=purego'])]:
                    subprocess.run(['go', 'test', '-count=1'] + flags + ['./...'], cwd=dest, check=True)
                    checks.append({'backend': backend, 'edition': edition, 'mode': mode, 'variant': variant})
            if file_sizes['CODE_SIZE'] >= file_sizes['SPEED']:
                raise RuntimeError('CODE_SIZE did not reduce generated source for ' + backend + '/' + edition)
            sizes.append({'backend': backend, 'edition': edition, 'source_bytes': file_sizes})
    negative = out / 'negative'
    negative.mkdir()
    for backend in BACKENDS:
        for option in ['marshaler', 'unmarshaler', 'sizer', 'protosizer']:
            value = 'true' if option == 'protosizer' else 'false'
            source = ('edition="2024";package bad;import option "google/protobuf/go_features.proto";import option "gogoproto/gogo.proto";'
                'option features.(pb.go).optimize_mode=CODE_SIZE;message Bad {option (gogoproto.' + option + ')=' + value + ';int32 value=1;}')
            (negative / 'bad.proto').write_text(source)
            process = subprocess.run(['protoc', '-I' + str(negative), '-I' + str(ROOT), '-I' + str(ROOT / 'protobuf'),
                '--plugin=protoc-gen-test=' + str(out / backend), '--test_out=' + str(negative), 'bad.proto'],
                stdout=subprocess.PIPE, stderr=subprocess.STDOUT, text=True)
            (negative / (backend + '-' + option + '.log')).write_text(process.stdout)
            expected = 'requires Size rather than protosizer' if option == 'protosizer' else 'conflicts with disabled codec'
            if process.returncode == 0 or expected not in process.stdout:
                raise RuntimeError('inconsistent optimization codec not diagnosed: ' + process.stdout)
    (out / 'qualification.json').write_text(json.dumps({'compiler': compiler, 'checks': checks, 'generated_sizes': sizes,
        'negative_checks': 20, 'status': 'passed'}, indent=2) + '\n')
    print('Optimization modes passed:', len(checks), 'consumer runs and 20 diagnostics')


if __name__ == '__main__':
    main()
