#!/usr/bin/env python3
"""Generate the unmodified upstream 36.2 conformance schemas in a temporary module."""
import argparse
import pathlib
import shutil
import subprocess

ROOT = pathlib.Path(__file__).resolve().parents[2]
SCHEMAS = {
    'protocol': 'conformance/conformance.proto',
    'proto2': 'google/protobuf/test_messages_proto2.proto',
    'proto3': 'google/protobuf/test_messages_proto3.proto',
    'editionproto2': 'editions/golden/test_messages_proto2_editions.proto',
    'editionproto3': 'editions/golden/test_messages_proto3_editions.proto',
    'edition2023': 'conformance/test_protos/test_messages_edition2023.proto',
}
MODULE = 'example.com/gogo-conformance'


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('source', type=pathlib.Path)
    parser.add_argument('output', type=pathlib.Path)
    parser.add_argument('--merge-oneof-messages', action='store_true')
    args = parser.parse_args()
    source, output = args.source.resolve(), args.output.resolve()
    output.mkdir(parents=True, exist_ok=True)
    driver = output / 'generator'
    subprocess.run(['go', 'build', '-o', str(driver), './protoc-gen-gogo'], cwd=ROOT, check=True)
    mappings = ['M' + path + '=' + MODULE + '/pb/' + key for key, path in SCHEMAS.items()]
    mappings += ['Mgoogle/protobuf/' + name + '.proto=github.com/gogo/protobuf/types'
                 for name in ['any', 'duration', 'empty', 'field_mask', 'struct', 'timestamp', 'wrappers']]
    if args.merge_oneof_messages:
        mappings.append('merge_oneof_messages=true')
    for key, path in SCHEMAS.items():
        stage = output / 'stage'
        stage.mkdir()
        subprocess.run(['protoc', '-I' + str(source), '-I' + str(source / 'src'),
                        '--plugin=protoc-gen-test=' + str(driver),
                        '--test_out=' + ','.join(['paths=source_relative'] + mappings) + ':' + str(stage), path], check=True)
        generated = list(stage.rglob('*.pb.go'))
        if len(generated) != 1:
            raise RuntimeError('expected one generated file for ' + path)
        dest = output / 'pb' / key
        dest.mkdir(parents=True)
        shutil.move(str(generated[0]), str(dest / 'messages.pb.go'))
        shutil.rmtree(stage)
    for name in ['main.go', 'main_test.go']:
        shutil.copyfile(ROOT / 'compat/conformance' / name, output / name)
    (output / 'go.mod').write_text('module ' + MODULE + '\n\ngo 1.15\n\nrequire github.com/gogo/protobuf v1.3.2\nreplace github.com/gogo/protobuf => ' + str(ROOT) + '\n')
    shutil.copyfile(ROOT / 'go.sum', output / 'go.sum')
    subprocess.run(['go', 'test', './...'], cwd=output, check=True)
    subprocess.run(['go', 'build', '-o', str(output / 'testee'), '.'], cwd=output, check=True)


if __name__ == '__main__':
    main()
