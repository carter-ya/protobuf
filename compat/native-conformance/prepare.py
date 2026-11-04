#!/usr/bin/env python3
"""Generate the unmodified upstream proto3 all-types schema with native HYBRID."""
import argparse
import pathlib
import shutil
import subprocess

ROOT = pathlib.Path(__file__).resolve().parents[2]
MODULE = 'example.com/native-conformance'


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('source', type=pathlib.Path)
    parser.add_argument('output', type=pathlib.Path)
    parser.add_argument('--official-generator', type=pathlib.Path)
    args = parser.parse_args()
    out = args.output.resolve()
    out.mkdir(parents=True)
    source = args.source.resolve()
    driver = out / 'generator'
    subprocess.run(['go', 'build', '-o', driver, './protoc-gen-gogo'], cwd=ROOT, check=True)
    schemas = {'alltypes': 'google/protobuf/test_messages_proto3.proto',
               'protocol': 'conformance/conformance.proto'}
    params = ['paths=source_relative', 'default_api_level=API_HYBRID']
    params += ['M' + path + '=' + MODULE + '/pb/' + key for key, path in schemas.items()]
    well_known = ['any', 'duration', 'empty', 'field_mask', 'struct', 'timestamp', 'wrappers']
    params += ['Mgoogle/protobuf/' + name + '.proto=github.com/gogo/protobuf/types' for name in well_known]
    for key, path in schemas.items():
        stage = out / 'stage'
        stage.mkdir()
        selected, options = driver, params
        if key == 'alltypes' and args.official_generator:
            selected = args.official_generator.resolve()
            options = [p for p in params if not p.startswith('Mgoogle/protobuf/')
                       or p.split('=')[0][1:] in schemas.values()]
        subprocess.run(['protoc', '-I' + str(source), '-I' + str(source / 'src'),
                        '--plugin=protoc-gen-test=' + str(selected),
                        '--test_out=' + ','.join(options) + ':' + str(stage), path], check=True)
        files = [f for f in stage.rglob('*.pb.go') if not f.name.endswith('_protoopaque.pb.go')]
        if len(files) != 1:
            raise RuntimeError('expected one non-opaque generated binding for ' + path)
        dest = out / 'pb' / key
        dest.mkdir(parents=True)
        shutil.move(files[0], dest / 'messages.pb.go')
        shutil.rmtree(stage)
    (out / 'go.mod').write_text('module ' + MODULE + '\n\ngo 1.23\nrequire (\n'
        ' github.com/gogo/protobuf v1.3.2\n github.com/gogo/protobuf/reflect v0.0.0\n'
        ' google.golang.org/protobuf v1.36.13-0.20261007182434-0c373ff94f29\n)\n'
        'replace github.com/gogo/protobuf => ' + str(ROOT) + '\n'
        'replace github.com/gogo/protobuf/reflect => ' + str(ROOT / 'reflect') + '\n')
    shutil.copyfile(ROOT / 'reflect/go.sum', out / 'go.sum')
    text = (ROOT / 'compat/native-conformance/main.go.txt').read_text()
    if args.official_generator:
        text = text.replace('live "github.com/gogo/protobuf/reflect"',
                            '"google.golang.org/protobuf/reflect/protoregistry"')
        text = text.replace('live.NewRegistry(n)',
                            'func()(*protoregistry.Types,error){return protoregistry.GlobalTypes,nil}()')
    (out / 'main.go').write_text(text)
    subprocess.run(['go', 'build', '-mod=mod', '-o', out / 'testee', '.'], cwd=out, check=True)


if __name__ == '__main__':
    main()
