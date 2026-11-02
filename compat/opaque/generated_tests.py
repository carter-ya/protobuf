#!/usr/bin/env python3
"""Compile and run generated gadget tests in both HYBRID build modes."""
import argparse
import json
import pathlib
import sys

ROOT = pathlib.Path(__file__).resolve().parents[2]
sys.path.insert(0, str(ROOT / 'compat/hybrid'))
from prepare import BACKENDS, DEFAULT_RUNTIME, module, run

SOURCE = '''syntax="proto3"; package gadget;
option go_package="example.com/native/pb";
import "gogoproto/gogo.proto";
option (gogoproto.testgen_all)=true;
option (gogoproto.equal_all)=true;
option (gogoproto.populate_all)=true;
option (gogoproto.marshaler_all)=true;
option (gogoproto.unmarshaler_all)=true;
option (gogoproto.sizer_all)=true;
message Record { int32 count=1; string label=2; repeated int32 values=3; }
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
        (dest / 'pb').mkdir(parents=True)
        (dest / 'gadget.proto').write_text(SOURCE)
        driver = dest / 'driver'
        run(['go', 'build', '-o', driver, './protoc-gen-' + backend])
        run(['protoc', '-I' + str(dest), '-I' + str(ROOT), '-I' + str(ROOT / 'protobuf'),
             '--plugin=protoc-gen-test=' + str(driver), '--test_out=paths=source_relative,default_api_level=API_HYBRID:' + str(dest / 'pb'), 'gadget.proto'])
        files = {p.name: p.read_text() for p in (dest / 'pb').glob('*.go')}
        for name, expression in [('gadgetpb_test.go', '!protoopaque'), ('gadget_protoopaquepb_test.go', 'protoopaque')]:
            if name not in files or not files[name].startswith('//go:build ' + expression + '\n'):
                raise RuntimeError('missing mutually exclusive generated test output: ' + name)
        module(dest, DEFAULT_RUNTIME)
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
