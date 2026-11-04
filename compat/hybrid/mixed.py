#!/usr/bin/env python3
"""Import native HYBRID packages from different backends in one process."""
import argparse
import pathlib
import subprocess

from prepare import DEFAULT_RUNTIME, module

ROOT = pathlib.Path(__file__).resolve().parents[2]


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('output', type=pathlib.Path)
    parser.add_argument('--runtime-version', default=DEFAULT_RUNTIME)
    args = parser.parse_args()
    out = args.output.resolve()
    out.mkdir(parents=True)
    (out / 'meta.proto').write_text(
        'syntax="proto2"; package metainfo; import "google/protobuf/descriptor.proto"; '
        'option go_package="example.com/meta"; '
        'extend google.protobuf.FileOptions { optional string label=51001; }')
    for name, backend in [('first', 'gogo'), ('second', 'gogofaster')]:
        driver = out / backend
        subprocess.run(['go', 'build', '-o', driver, './protoc-gen-' + backend], cwd=ROOT, check=True)
        (out / (name + '.proto')).write_text(
            'edition="2024"; package ' + name + '; import option "meta.proto"; '
            'import option "google/protobuf/go_features.proto"; '
            'option go_package="example.com/native/' + name + '"; '
            'option features.(pb.go).api_level=API_HYBRID; '
            'option (metainfo.label)="same"; message Record { string value=1; }')
        stage = out / name
        stage.mkdir()
        subprocess.run(['protoc', '-I' + str(out), '--plugin=protoc-gen-test=' + str(driver),
                        '--test_out=paths=source_relative:' + str(stage), name + '.proto'], check=True)
    module(out, args.runtime_version)
    (out / 'mixed_test.go').write_text('''package consumer
import ("testing"; first "example.com/native/first"; second "example.com/native/second")
func TestMixed(t *testing.T) {
 a:=new(first.Record);b:=new(second.Record);a.SetValue("a");b.SetValue("b")
 if a.ProtoReflect().Descriptor().FullName()==b.ProtoReflect().Descriptor().FullName(){t.Fatal("descriptor identity")}
}
''')
    subprocess.run(['go', 'test', '-mod=mod', '-race', './...'], cwd=out, check=True)


if __name__ == '__main__':
    main()
