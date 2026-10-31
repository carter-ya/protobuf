#!/usr/bin/env python3
"""Qualify opt-in oneof message merging independently of legacy generation."""
import argparse
import pathlib
import shutil
import subprocess

ROOT = pathlib.Path(__file__).resolve().parents[2]
BACKENDS = ['gogo', 'gogofast', 'gogofaster', 'gogoslick', 'gogotypes']
SCHEMA = '''syntax = "SYNTAX";
package oneofmerge;
option go_package = "example.com/oneof-merge/pb";
message Child { LABEL int32 a = 1; LABEL int32 b = 2; }
message Root { oneof choice { Child child = 1; Child replacement = 2; } LABEL int32 extra = 3; }
'''


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('output', type=pathlib.Path)
    parser.add_argument('--backend', action='append')
    parser.add_argument('--scalar-members', choices=['auto', 'nullable'])
    args = parser.parse_args()
    output = args.output.resolve()
    output.mkdir(parents=True)
    for backend in args.backend or BACKENDS:
        driver = output / backend
        subprocess.run(['go', 'build', '-o', str(driver), './protoc-gen-' + backend], cwd=ROOT, check=True)
        for syntax in ['proto2', 'proto3']:
            for mode in ['legacy', 'standard']:
                dest = output / (syntax + '-' + mode + '-' + backend)
                print(dest.name, flush=True)
                (dest / 'pb').mkdir(parents=True)
                schema = SCHEMA.replace('SYNTAX', syntax).replace('LABEL', 'optional' if syntax == 'proto2' else '')
                if args.scalar_members:
                    schema = schema.replace('package oneofmerge;', 'import "gogoproto/gogo.proto";\npackage oneofmerge;')
                    schema += '\noption (gogoproto.equal_all) = true;\noption (gogoproto.verbose_equal_all) = true;\noption (gogoproto.compare_all) = true;\noption (gogoproto.populate_all) = true;\noption (gogoproto.gostring_all) = true;\n'
                    scalar = 'int32 number = 2;'
                    if args.scalar_members == 'nullable':
                        scalar = 'int32 number = 2 [(gogoproto.nullable) = true];'
                    schema = schema.replace('Child replacement = 2;', scalar)
                (dest / 'sample.proto').write_text(schema)
                parameters = 'paths=source_relative,merge_oneof_messages=' + ('true' if mode == 'standard' else 'false')
                subprocess.run(['protoc', '-I' + str(dest), '-I' + str(ROOT), '-I' + str(ROOT / 'protobuf'), '--plugin=protoc-gen-test=' + str(driver),
                                '--test_out=' + parameters + ':' + str(dest / 'pb'), 'sample.proto'], check=True)
                code = (dest / 'pb/sample.pb.go').read_text()
                if mode == 'standard' and ('GoGoProtoPackageIsVersion12' not in code or 'oneof_merge' not in code):
                    raise RuntimeError('opt-in oneof metadata or runtime guard missing')
                if mode == 'legacy' and ('GoGoProtoPackageIsVersion12' in code or 'oneof_merge' in code):
                    raise RuntimeError('legacy generation changed')
                (dest / 'go.mod').write_text('module example.com/oneof-merge\ngo 1.15\nrequire github.com/gogo/protobuf v1.3.2\nreplace github.com/gogo/protobuf => ' + str(ROOT) + '\n')
                shutil.copyfile(ROOT / 'go.sum', dest / 'go.sum')
                test = (ROOT / 'compat/oneof-merge/consumer_test.go.txt').read_text()
                if args.scalar_members:
                    test = test.replace('import (', 'import (\n"math/rand"')
                    test = test.replace('10, 2, 8, 1, 18, 0, 10, 2, 16, 2', '10, 2, 8, 1, 16, 0, 10, 2, 16, 2')
                    test += (ROOT / 'compat/oneof-merge/scalar_test.go.txt').read_text()
                (dest / 'consumer_test.go').write_text(test.replace('MERGE', 'true' if mode == 'standard' else 'false').replace('FAST', 'false' if backend == 'gogo' else 'true'))
                subprocess.run(['go', 'test', './...'], cwd=dest, check=True)
                subprocess.run(['go', 'test', '-tags=purego', './...'], cwd=dest, check=True)


if __name__ == '__main__':
    main()
