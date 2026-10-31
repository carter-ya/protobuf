#!/usr/bin/env python3
"""Prepare five isolated Go fuzz modules from pinned and local wire fixtures."""
import argparse
import pathlib
import shutil
import subprocess

ROOT = pathlib.Path(__file__).resolve().parents[3]
MODULE = 'example.com/gogo-wire-qualification'
OFFICIAL = 'conformance/test_protos/test_messages_edition2023.proto'
CLOSED = 'compat/qualify/binary/closed.proto'
BACKENDS = ('gogo', 'gogofast', 'gogofaster', 'gogoslick', 'gogotypes')


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('source', type=pathlib.Path)
    parser.add_argument('output', type=pathlib.Path)
    args = parser.parse_args()
    source, output = args.source.resolve(), args.output.resolve()
    output.mkdir(parents=True, exist_ok=True)
    subprocess.run(['protoc', '-I' + str(ROOT), '-I' + str(source),
                    '-I' + str(source / 'src'), '--include_imports',
                    '--descriptor_set_out=' + str(output / 'schema.pb'),
                    OFFICIAL, CLOSED], check=True)
    for backend in BACKENDS:
        module = output / backend
        module.mkdir(parents=True, exist_ok=True)
        driver = module / 'generator'
        subprocess.run(['go', 'build', '-o', str(driver), './protoc-gen-' + backend],
                       cwd=ROOT, check=True)
        mappings = ['M' + OFFICIAL + '=' + MODULE + '/official',
                    'M' + CLOSED + '=' + MODULE + '/closed']
        for schema, package in ((OFFICIAL, 'official'), (CLOSED, 'closed')):
            stage = module / 'stage'
            stage.mkdir()
            subprocess.run(['protoc', '-I' + str(ROOT), '-I' + str(source),
                            '-I' + str(source / 'src'),
                            '--plugin=protoc-gen-test=' + str(driver),
                            '--test_out=' + ','.join(['paths=source_relative',
                                                     'merge_oneof_messages=true'] + mappings)
                            + ':' + str(stage), schema], check=True)
            generated = list(stage.rglob('*.pb.go'))
            if len(generated) != 1:
                raise RuntimeError('expected one generated file for ' + schema)
            destination = module / package
            destination.mkdir(exist_ok=True)
            shutil.move(str(generated[0]), str(destination / 'messages.pb.go'))
            shutil.rmtree(stage)
        driver.unlink()
        shutil.copyfile(ROOT / 'compat/qualify/binary/wire_test.go.txt',
                        module / 'wire_test.go')
        (module / 'go.mod').write_text(
            'module ' + MODULE + '\n\ngo 1.23\n\nrequire (\n github.com/gogo/protobuf v1.3.2\n google.golang.org/protobuf v1.36.12\n)\n'
            'replace github.com/gogo/protobuf => ' + str(ROOT) + '\n')
        shutil.copyfile(ROOT / 'compat/qualify/go.sum', module / 'go.sum')
        corpus = ROOT / 'compat/qualify/binary/testdata'
        if corpus.exists():
            shutil.copytree(corpus, module / 'testdata', dirs_exist_ok=True)
        compiled = module / 'qualification.test'
        subprocess.run(['go', 'test', '-c', '-o', str(compiled), '.'], cwd=module, check=True)
        compiled.unlink()


if __name__ == '__main__':
    main()
