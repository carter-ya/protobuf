#!/usr/bin/env python3
"""Qualify HYBRID normal and protoopaque builds from the same generated outputs."""
import argparse
import json
import pathlib
import shutil
import sys

ROOT = pathlib.Path(__file__).resolve().parents[2]
HERE = pathlib.Path(__file__).resolve().parent
HYBRID = ROOT / 'compat/hybrid'
sys.path.insert(0, str(HYBRID))
from prepare import BACKENDS, DEFAULT_RUNTIME, module, run


def tagged(source, opaque):
    expression = 'protoopaque' if opaque else '!protoopaque'
    return '//go:build ' + expression + '\n// +build ' + expression + '\n\n' + source


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('output', type=pathlib.Path)
    parser.add_argument('--backend', action='append', choices=BACKENDS)
    parser.add_argument('--flavor', action='append', choices=['proto3', 'proto2', '2023', '2024', '2026'])
    parser.add_argument('--official-generator', type=pathlib.Path)
    args = parser.parse_args()
    out = args.output.resolve()
    out.mkdir(parents=True)
    oracle = None
    if args.official_generator:
        reference = out / 'official'
        reference.mkdir()
        run(['protoc', '-I.', '--plugin=protoc-gen-go=' + str(args.official_generator.resolve()),
             '--go_out=paths=source_relative,default_api_level=API_HYBRID:' + str(reference), 'reflect/internal/testpb/schema.proto'])
        oracle = out / 'compare-api'
        run(['go', 'build', '-o', oracle, './compat/hybrid/compare-api'])
    checks = []
    for backend in args.backend or BACKENDS:
        driver = out / backend
        run(['go', 'build', '-o', driver, './protoc-gen-' + backend])
        for flavor in args.flavor or ['proto3', 'proto2', '2023', '2024', '2026']:
            dest = out / (backend + '-' + flavor)
            (dest / 'pb').mkdir(parents=True)
            if flavor.isdigit():
                source = (HYBRID / 'edition.proto').read_text().replace('"2024"', '"' + flavor + '"')
                if flavor == '2023':
                    source = source.replace('import option ', 'import ').replace('option features.enforce_naming_style = STYLE_LEGACY;\n', '')
                if flavor == '2026':
                    source = source.replace('// JSON_IMPORT', 'import option "google/protobuf/json_enumvalue_options.proto";').replace('ONE = 1;  // JSON_VALUE', 'ONE = 1 [(pb.enumvalue.json).string = "named-one"];')
                (dest / 'edition.proto').write_text(source)
                run(['protoc', '-I' + str(dest), '--plugin=protoc-gen-test=' + str(driver),
                     '--test_out=paths=source_relative,annotate_code=true:' + str(dest / 'pb'), 'edition.proto'])
                normal = (HYBRID / 'edition_test.go.txt').read_text().replace('EDITION', flavor)
                opaque = normal.replace('&pb.Child{Name:gogo.String("child")}', 'pb.Child_builder{Name:gogo.String("child")}.Build()').replace('map[string]*pb.Child{"x":{Name:gogo.String("mapped")}}', 'map[string]*pb.Child{"x":pb.Child_builder{Name:gogo.String("mapped")}.Build()}')
                opaque += '\nfunc TestEditionOpaqueFields(t *testing.T){typ:=reflect.TypeOf(pb.Sample{});for i:=0;i<typ.NumField();i++{f:=typ.Field(i);if (f.Tag.Get("protobuf")!=""||f.Tag.Get("protobuf_oneof")!="")&&f.PkgPath==""{t.Fatal("exposed field",f.Name)}};if _,ok:=reflect.TypeOf(pb.Open{}).FieldByName("Value");!ok{t.Fatal("OPEN override hidden")}}\n'
            else:
                inputs = ['reflect/internal/testpb/schema.proto']
                if flavor == 'proto2':
                    inputs.append('reflect/internal/testpb/closed.proto')
                run(['protoc', '-I.', '-Iprotobuf', '--plugin=protoc-gen-test=' + str(driver),
                     '--test_out=paths=source_relative,annotate_code=true,default_api_level=API_HYBRID:' + str(dest / 'pb')] + inputs)
                for src in (dest / 'pb/reflect/internal/testpb').iterdir():
                    shutil.move(src, dest / 'pb' / src.name)
                shutil.rmtree(dest / 'pb/reflect')
                normal = (HYBRID / 'consumer_test.go.txt').read_text().replace('HYBRID', 'true')
                (dest / 'accessors_test.go').write_text(tagged((HYBRID / 'hybrid_test.go.txt').read_text(), False))
                opaque = (HERE / 'consumer_test.go.txt').read_text()
                if flavor == 'proto2':
                    closed = (HYBRID / 'closed_test.go.txt').read_text()
                    (dest / 'closed_test.go').write_text(tagged(closed, False))
                    shutil.copyfile(HERE / 'required_test.go.txt', dest / 'required_test.go')
                    (dest / 'closed_protoopaque_test.go').write_text(tagged(closed.replace('n.Status!=nil', 'n.HasStatus()').replace('n.Child.GetValue()', 'n.GetChild().GetValue()'), True))
                if oracle and flavor == 'proto3':
                    for name, flags in [('schema.pb.go', []), ('schema_protoopaque.pb.go', ['--opaque'])]:
                        run([oracle, reference / 'reflect/internal/testpb' / name, dest / 'pb' / name] + flags)
            normal_go = sorted((dest / 'pb').glob('*.pb.go'))
            if not any(p.name.endswith('_protoopaque.pb.go') for p in normal_go):
                raise RuntimeError('missing protoopaque alternate output')
            if not all((p.with_name(p.name + '.meta')).exists() for p in normal_go):
                raise RuntimeError('missing build-specific annotations')
            (dest / 'consumer_test.go').write_text(tagged(normal, False))
            (dest / 'consumer_protoopaque_test.go').write_text(tagged(opaque, True))
            module(dest, DEFAULT_RUNTIME)
            for mode, tags in [('hybrid', []), ('opaque', ['protoopaque'])]:
                for variant, flags in [('default', []), ('purego', ['purego']), ('race', [])]:
                    selected = tags + flags
                    command = ['go', 'test', '-mod=mod', '-count=1']
                    if selected:
                        command += ['-tags=' + ','.join(selected)]
                    if variant == 'race':
                        command += ['-race']
                    run(command + ['./...'], dest)
                    checks.append({'backend': backend, 'flavor': flavor, 'mode': mode, 'variant': variant})
    (out / 'qualification.json').write_text(json.dumps({'status': 'passed', 'checks': checks}, indent=2) + '\n')


if __name__ == '__main__':
    main()
