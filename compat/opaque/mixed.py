#!/usr/bin/env python3
"""Check file API defaults and imports across OPEN, HYBRID and OPAQUE packages."""
import argparse
import json
import pathlib
import shutil
import sys

ROOT = pathlib.Path(__file__).resolve().parents[2]
sys.path.insert(0, str(ROOT / 'compat/hybrid'))
from prepare import BACKENDS, DEFAULT_RUNTIME, module, run

TEST = '''package mixed
import (
 "reflect"
 "testing"
 legacy "example.com/mixed/legacy"
 modern "example.com/mixed/modern"
 opaque "example.com/mixed/opaque"
 gogo "github.com/gogo/protobuf/proto"
 live "github.com/gogo/protobuf/reflect"
 v2 "google.golang.org/protobuf/proto"
 "google.golang.org/protobuf/encoding/protojson"
 "google.golang.org/protobuf/testing/prototest"
)
func TestMixedFileSelections(t *testing.T) {
 if _,ok:=reflect.TypeOf(new(legacy.Leaf)).MethodByName("ProtoReflect");ok{t.Fatal("OPEN file gained native methods")}
 _,exported:=reflect.TypeOf(modern.Record{}).FieldByName("Value");if exported==opaqueBuild{t.Fatal("HYBRID selection did not follow build tag")}
 if _,ok:=reflect.TypeOf(opaque.Container{}).FieldByName("Child");ok{t.Fatal("direct OPAQUE file was opened")}
 leaf:=&legacy.Leaf{Label:"legacy"}
 record:=modern.Record_builder{Value:gogo.Int32(0),Child:leaf}.Build()
 n:=opaque.Container_builder{Child:record}.Build()
 child:=record.ProtoReflect().Get(record.ProtoReflect().Descriptor().Fields().ByName("child")).Message()
 native,ok:=live.Unwrap(child);if !ok||native!=leaf{t.Fatal("frozen imported child lost storage identity")}
 data,err:=v2.Marshal(n);if err!=nil{t.Fatal(err)};target:=new(opaque.Container)
 if err=gogo.Unmarshal(data,target);err!=nil||!gogo.Equal(n,target){t.Fatal("mixed binary",err)}
 data,err=protojson.Marshal(n);if err!=nil{t.Fatal(err)}
 if err=protojson.Unmarshal(data,target);err!=nil||!v2.Equal(n,target){t.Fatal("mixed JSON",err)}
 prototest.Message{}.Test(t,n.ProtoReflect().Type())
}
'''


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('output', type=pathlib.Path)
    parser.add_argument('--backend', action='append', choices=BACKENDS)
    args = parser.parse_args()
    out = args.output.resolve()
    out.mkdir(parents=True)
    checks = []
    for backend in args.backend or BACKENDS:
        dest = out / backend
        generated = dest / 'generated'
        generated.mkdir(parents=True)
        driver = dest / 'driver'
        run(['go', 'build', '-o', driver, './protoc-gen-' + backend])
        sources = {
            'legacy': 'import "gogoproto/gogo.proto"; option (gogoproto.goproto_unrecognized_all)=true; message Leaf {string label=1;}',
            'modern': 'import "legacy.proto"; message Record {optional int32 value=1; legacy.Leaf child=2;}',
            'opaque': 'import "modern.proto"; message Container {modern.Record child=1;}',
        }
        for name, source in sources.items():
            (dest / (name + '.proto')).write_text('syntax="proto3"; package ' + name + '; option go_package="example.com/mixed/' + name + '"; ' + source)
        for name in sources:
            run(['protoc', '-I' + str(dest), '-I' + str(ROOT), '-I' + str(ROOT / 'protobuf'),
                 '--plugin=protoc-gen-test=' + str(driver),
                 '--test_out=paths=source_relative,default_api_level=API_HYBRID,apilevelMlegacy.proto=API_OPEN,apilevelMopaque.proto=API_OPAQUE:' + str(generated), name + '.proto'])
        for name in sources:
            package = dest / name
            package.mkdir()
            for file in generated.glob(name + '*.go'):
                shutil.move(file, package / file.name)
        if (dest / 'legacy/legacy_protoopaque.pb.go').exists() or (dest / 'opaque/opaque_protoopaque.pb.go').exists():
            raise RuntimeError('non-HYBRID file unexpectedly has alternate output')
        if not (dest / 'modern/modern_protoopaque.pb.go').exists():
            raise RuntimeError('HYBRID file missing alternate output')
        (dest / 'consumer_test.go').write_text(TEST)
        for opaque, expression in [(False, '!protoopaque'), (True, 'protoopaque')]:
            (dest / ('mode_' + str(opaque).lower() + '_test.go')).write_text('//go:build ' + expression + '\n// +build ' + expression + '\n\npackage mixed\nconst opaqueBuild=' + str(opaque).lower() + '\n')
        module(dest, DEFAULT_RUNTIME)
        gomod = dest / 'go.mod'
        gomod.write_text(gomod.read_text().replace('module example.com/native', 'module example.com/mixed'))
        for mode, tags in [('hybrid', []), ('opaque', ['protoopaque'])]:
            for variant, flags in [('default', []), ('purego', ['purego']), ('race', [])]:
                command = ['go', 'test', '-mod=mod', '-count=1']
                if tags + flags:
                    command += ['-tags=' + ','.join(tags + flags)]
                if variant == 'race':
                    command += ['-race']
                run(command + ['./...'], dest)
                checks.append({'backend': backend, 'mode': mode, 'variant': variant})
    (out / 'qualification.json').write_text(json.dumps({'status': 'passed', 'checks': checks}, indent=2) + '\n')


if __name__ == '__main__':
    main()
