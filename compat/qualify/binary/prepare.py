#!/usr/bin/env python3
"""Prepare five isolated Go fuzz modules from pinned and local wire fixtures."""
import argparse
import json
import os
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
    parser.add_argument('--byte-extensions', action='store_true')
    parser.add_argument('--optimize-mode', choices=['SPEED', 'CODE_SIZE'], default=os.environ.get('BINARY_OPTIMIZE_MODE') or None)
    args = parser.parse_args()
    source, output = args.source.resolve(), args.output.resolve()
    output.mkdir(parents=True, exist_ok=True)
    if args.optimize_mode not in [None, 'SPEED', 'CODE_SIZE']:
        parser.error('invalid BINARY_OPTIMIZE_MODE')
    includes = []
    if args.byte_extensions or args.optimize_mode:
        shadow = output / 'inputs'
        fixture = shadow / CLOSED
        fixture.parent.mkdir(parents=True, exist_ok=True)
        text = (ROOT / CLOSED).read_text()
        if args.byte_extensions:
            text = text.replace('package qualification.closed;',
                'package qualification.closed;\n\nimport "gogoproto/gogo.proto";\n'
                'option (gogoproto.goproto_extensions_map_all) = false;')
        if args.optimize_mode:
            # The Go feature starts in 2024. Keep every wire feature identical
            # to the native C++ 2023 fixture; naming/Go optimization are not wire
            # semantics. The official all-types 2023 schema stays an unmarked
            # control alongside the selected local fixture.
            text = text.replace('edition = "2023";', 'edition = "2024";')
            text = text.replace('import "gogoproto/gogo.proto";', 'import option "gogoproto/gogo.proto";')
            text = text.replace('package qualification.closed;',
                'package qualification.closed;\nimport option "google/protobuf/go_features.proto";\n'
                'option features.enforce_naming_style = STYLE_LEGACY;\n'
                'option features.(pb.go).optimize_mode = ' + args.optimize_mode + ';')
        fixture.write_text(text)
        includes = ['-I' + str(shadow)]
    (output / 'profile.json').write_text(json.dumps({
        'optimize_mode': args.optimize_mode, 'byte_extensions': args.byte_extensions,
        'compiler': subprocess.check_output(['protoc', '--version'], text=True).strip(),
        'source_head': subprocess.check_output(['git', 'rev-parse', 'HEAD'], cwd=ROOT, text=True).strip(),
        'working_tree': subprocess.check_output(['git', 'status', '--porcelain'], cwd=ROOT, text=True),
        'closed_edition': 2024 if args.optimize_mode else 2023,
        'official_edition': 2023, 'backends': BACKENDS,
    }, indent=2) + '\n')
    subprocess.run(['protoc'] + includes + ['-I' + str(ROOT), '-I' + str(source),
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
            subprocess.run(['protoc'] + includes + ['-I' + str(ROOT), '-I' + str(source),
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
        guard = 'GoGoProtoPackageIsVersion16' if args.optimize_mode else 'GoGoProtoPackageIsVersion14'
        if guard not in (module / 'closed/messages.pb.go').read_text():
            raise RuntimeError('registered Editions extension runtime guard missing')
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
