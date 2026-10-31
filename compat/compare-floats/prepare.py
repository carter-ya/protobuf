#!/usr/bin/env python3
"""Qualify generated floating-point comparison across backends and syntaxes."""
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
        parser.error('use a pinned qualified compiler: 3.21.8, 36.2 or 37.0-rc1')
    syntaxes = ['proto2', 'proto3'] + ([] if version == 'libprotoc 3.21.8' else ['2023', '2024', '2026'])
    output = args.output.resolve()
    output.mkdir(parents=True)
    template = (ROOT / 'compat/compare-floats/float.proto').read_text()
    for backend in BACKENDS:
        driver = output / backend
        subprocess.run(['go', 'build', '-o', str(driver), './protoc-gen-' + backend], cwd=ROOT, check=True)
        for syntax in syntaxes:
            module = output / (syntax + '-' + backend)
            package = module / 'pb'
            package.mkdir(parents=True)
            source = template.replace('"2023"', '"' + syntax + '"')
            if syntax.startswith('proto'):
                source = source.replace('edition = "' + syntax + '";', 'syntax = "' + syntax + '";')
                source = source.replace(' [features.field_presence = IMPLICIT]', '')
                if syntax == 'proto2':
                    source = re.sub(r'^  (float|double|int32) ', r'  optional \1 ', source, flags=re.MULTILINE)
                else:
                    source = re.sub(r'^  (float|double) explicit_', r'  optional \1 explicit_', source, flags=re.MULTILINE)
            (module / 'float.proto').write_text(source)
            subprocess.run(['protoc', '-I' + str(module), '-I' + str(ROOT),
                '--plugin=protoc-gen-test=' + str(driver), '--test_out=paths=source_relative:' + str(package), 'float.proto'], check=True)
            (module / 'consumer_test.go').write_text((ROOT / 'compat/compare-floats/consumer_test.go.txt').read_text())
            run(module)
    # Qualify freshly generated WKT methods as well as the maintained bindings.
    module = output / 'wrappers'
    stage = module / 'stage'
    stage.mkdir(parents=True)
    subprocess.run(['protoc', '-I' + str(ROOT / 'protobuf'), '-I' + str(ROOT),
        '--plugin=protoc-gen-test=' + str(output / 'gogotypes'), '--test_out=paths=source_relative:' + str(stage),
        'google/protobuf/wrappers.proto'], check=True)
    (module / 'pb').mkdir()
    shutil.move(str(stage / 'google/protobuf/wrappers.pb.go'), str(module / 'pb/wrappers.pb.go'))
    shutil.rmtree(stage)
    test = (ROOT / 'types/compare_nan_test.go').read_text().replace('package types_test', 'package qualify').replace(
        'github.com/gogo/protobuf/types', 'example.com/gogo-float-compare/pb')
    (module / 'consumer_test.go').write_text(test)
    run(module)
    print('Qualified', len(BACKENDS) * len(syntaxes), 'generated profiles plus fresh WKT wrappers')


def run(module):
    (module / 'go.mod').write_text('module example.com/gogo-float-compare\n\ngo 1.15\nrequire github.com/gogo/protobuf v1.3.2\nreplace github.com/gogo/protobuf => ' + str(ROOT) + '\n')
    shutil.copyfile(ROOT / 'go.sum', module / 'go.sum')
    subprocess.run(['go', 'test', './...'], cwd=module, check=True)
    subprocess.run(['go', 'test', '-tags=purego', './...'], cwd=module, check=True)


if __name__ == '__main__':
    main()
