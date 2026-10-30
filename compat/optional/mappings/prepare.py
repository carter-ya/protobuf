#!/usr/bin/env python3
"""Generate independent optional-mapping consumers for every public backend."""
import argparse
import pathlib
import shutil
import subprocess

ROOT = pathlib.Path(__file__).resolve().parents[3]
BACKENDS = ['gogo', 'gogofast', 'gogofaster', 'gogoslick', 'gogotypes']


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('output', type=pathlib.Path)
    parser.add_argument('--backend', action='append')
    args = parser.parse_args()
    output = args.output.resolve()
    output.mkdir(parents=True)
    for backend in args.backend or BACKENDS:
        driver = output / backend
        subprocess.run(['go', 'build', '-o', str(driver), './protoc-gen-' + backend], cwd=ROOT, check=True)
        dest = output / ('cast-' + backend)
        (dest / 'pb').mkdir(parents=True)
        (dest / 'types').mkdir()
        shutil.copyfile(ROOT / 'compat/optional/mappings/cast.proto', dest / 'sample.proto')
        shutil.copyfile(ROOT / 'compat/optional/mappings/types.go.txt', dest / 'types/types.go')
        (dest / 'gogoproto').mkdir()
        shutil.copyfile(ROOT / 'gogoproto/gogo.proto', dest / 'gogoproto/gogo.proto')
        subprocess.run(['protoc', '-I' + str(dest), '-I' + str(ROOT / 'protobuf'),
                        '--plugin=protoc-gen-test=' + str(driver),
                        '--test_out=paths=source_relative:' + str(dest / 'pb'), 'sample.proto'], check=True)
        code = (dest / 'pb/sample.pb.go').read_text()
        if 'GoGoProtoPackageIsVersion10' not in code:
            raise RuntimeError('optional mapping runtime guard missing')
        (dest / 'go.mod').write_text('module example.com/optional-mappings\ngo 1.15\nrequire github.com/gogo/protobuf v1.3.2\nreplace github.com/gogo/protobuf => ' + str(ROOT) + '\n')
        shutil.copyfile(ROOT / 'go.sum', dest / 'go.sum')
        test = (ROOT / 'compat/optional/mappings/consumer_test.go.txt').read_text()
        (dest / 'consumer_test.go').write_text(test.replace('FAST', 'false' if backend == 'gogo' else 'true'))
        subprocess.run(['go', 'test', './...'], cwd=dest, check=True)
        subprocess.run(['go', 'test', '-tags=purego', './...'], cwd=dest, check=True)


if __name__ == '__main__':
    main()
