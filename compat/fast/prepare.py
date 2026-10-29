#!/usr/bin/env python3
import argparse
import pathlib
import shutil
import subprocess

ROOT = pathlib.Path(__file__).resolve().parents[2]
BACKENDS = ['gogo', 'gogofast', 'gogofaster', 'gogoslick', 'gogotypes']


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('output', type=pathlib.Path)
    args = parser.parse_args()
    output = args.output.resolve()
    output.mkdir(parents=True)
    for backend in BACKENDS:
        subprocess.run(['go', 'build', '-o', str(output / backend), './protoc-gen-' + backend], cwd=ROOT, check=True)
    source = (ROOT / 'compat/fast/scalars.proto').read_text()
    for edition in ['2023', '2024', '2026']:
        for backend in BACKENDS:
            dest = output / (edition + '-' + backend)
            dest.mkdir()
            (dest / 'pb').mkdir()
            (dest / 'scalars.proto').write_text(source.replace('"2023"', '"' + edition + '"'))
            subprocess.run(['protoc', '-I' + str(dest), '--plugin=protoc-gen-test=' + str(output / backend), '--test_out=paths=source_relative:' + str(dest / 'pb'), 'scalars.proto'], check=True)
            code = (dest / 'pb/scalars.pb.go').read_text()
            if backend != 'gogo' and 'GoGoProtoPackageIsVersion8' not in code:
                raise RuntimeError('fast runtime guard missing')
            (dest / 'go.mod').write_text('module example.com/gogo-fast\ngo 1.15\nrequire github.com/gogo/protobuf v1.3.2\nreplace github.com/gogo/protobuf => ' + str(ROOT) + '\n')
            shutil.copyfile(ROOT / 'go.sum', dest / 'go.sum')
            test = (ROOT / 'compat/fast/scalars_test.go.txt').read_text().replace('FAST', 'true' if backend != 'gogo' else 'false')
            (dest / 'scalars_test.go').write_text(test)
            subprocess.run(['go', 'test', './...'], cwd=dest, check=True)


if __name__ == '__main__':
    main()
