#!/usr/bin/env python3
import argparse
import pathlib
import shutil
import subprocess

ROOT = pathlib.Path(__file__).resolve().parents[2]


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('output', type=pathlib.Path)
    args = parser.parse_args()
    output = args.output.resolve()
    output.mkdir(parents=True)
    gogo, gostring = output / 'gogo', output / 'gostring'
    for target, package in [(gogo, './protoc-gen-gogo'), (gostring, './protoc-gen-gostring')]:
        subprocess.run(['go', 'build', '-o', str(target), package], cwd=ROOT, check=True)
    original = (ROOT / 'compat/plugins/sample.proto').read_text()
    for edition in ['2023', '2024', '2026']:
        for auxiliary in [False, True]:
            dest = output / (edition + ('-aux' if auxiliary else ''))
            dest.mkdir()
            (dest / 'pb').mkdir()
            source = original.replace('"2023"', '"' + edition + '"')
            if auxiliary:
                source = source.replace('(gogoproto.gostring_all) = true', '(gogoproto.gostring_all) = false')
            (dest / 'sample.proto').write_text(source)
            base = ['protoc', '-I' + str(dest), '-I' + str(ROOT), '-I' + str(ROOT / 'protobuf')]
            subprocess.run(base + ['--plugin=protoc-gen-test=' + str(gogo), '--test_out=paths=source_relative:' + str(dest / 'pb'), 'sample.proto'], check=True)
            if auxiliary:
                subprocess.run(base + ['--plugin=protoc-gen-test=' + str(gostring), '--test_out=paths=source_relative:' + str(dest / 'pb'), 'sample.proto'], check=True)
            (dest / 'go.mod').write_text('module example.com/gogo-plugins\ngo 1.15\nrequire github.com/gogo/protobuf v1.3.2\nreplace github.com/gogo/protobuf => ' + str(ROOT) + '\n')
            shutil.copyfile(ROOT / 'go.sum', dest / 'go.sum')
            shutil.copyfile(ROOT / 'compat/plugins/consumer_test.go.txt', dest / 'consumer_test.go')
            subprocess.run(['go', 'test', './...'], cwd=dest, check=True)
    for option in ['(gogoproto.marshaler_all) = true', '(gogoproto.goproto_unrecognized_all) = false', '(gogoproto.face_all) = true']:
        source = original + '\noption ' + option + ';\n'
        (output / 'bad.proto').write_text(source)
        process = subprocess.run(['protoc', '-I' + str(output), '-I' + str(ROOT), '-I' + str(ROOT / 'protobuf'), '--plugin=protoc-gen-test=' + str(gogo), '--test_out=' + str(output), 'bad.proto'], capture_output=True, text=True)
        if process.returncode == 0 or ('gogo' not in process.stderr):
            raise RuntimeError('unsupported option accepted: ' + option + '\n' + process.stderr)


if __name__ == '__main__':
    main()
