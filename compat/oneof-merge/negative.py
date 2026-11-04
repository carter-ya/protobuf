#!/usr/bin/env python3
"""Check opt-in diagnostics and unchanged default generation."""
import pathlib
import subprocess
import sys

ROOT = pathlib.Path(__file__).resolve().parents[2]
output = pathlib.Path(sys.argv[1]).resolve()
driver = output / 'gogo'
base = '''syntax = "proto3";
import "gogoproto/gogo.proto";
package oneofnegative;
option go_package = "example.com/oneof-negative";
message Child { int32 a = 1; }
'''
cases = {
    'custom': (base + 'message Root { oneof choice { Child child = 1 [(gogoproto.customtype) = "example.com/types.Payload"]; } }', 'true', 'does not support custom, cast or standard message mappings'),
    'cast': (base + 'message Root { oneof choice { Child child = 1 [(gogoproto.casttype) = "example.com/types.Payload"]; } }', 'true', 'does not support custom, cast or standard message mappings'),
    'std': (base + 'import "google/protobuf/timestamp.proto"; message Root { oneof choice { google.protobuf.Timestamp child = 1 [(gogoproto.stdtime) = true]; } }', 'true', 'does not support custom, cast or standard message mappings'),
    'foreign-runtime': (base + 'option (gogoproto.gogoproto_import) = false; message Root { oneof choice { Child child = 1; } }', 'true', 'requires the gogo runtime'),
    'invalid-boolean': (base + 'message Root { oneof choice { Child child = 1; } }', 'invalid', 'must be true or false'),
}

def generate(name, schema, parameter):
    dest = output / ('negative-' + name)
    dest.mkdir()
    (dest / 'sample.proto').write_text(schema)
    result = subprocess.run(['protoc', '-I' + str(dest), '-I' + str(ROOT), '-I' + str(ROOT / 'protobuf'),
                             '--plugin=protoc-gen-test=' + str(driver), '--test_out=paths=source_relative' + parameter + ':' + str(dest), 'sample.proto'], text=True, capture_output=True)
    return result, dest

for name, (schema, value, message) in cases.items():
    result, _ = generate(name, schema, ',merge_oneof_messages=' + value)
    if result.returncode == 0 or message not in result.stderr:
        raise RuntimeError(name + ': unexpected diagnostic: ' + result.stderr)

schema = base + 'message Root { oneof choice { Child child = 1; } }'
default, first = generate('default', schema, '')
legacy, second = generate('false', schema, ',merge_oneof_messages=false')
if default.returncode or legacy.returncode or (first / 'sample.pb.go').read_bytes() != (second / 'sample.pb.go').read_bytes():
    raise RuntimeError('false parameter changed default generation')
scalar, third = generate('scalar-only', base + 'message Root { oneof choice { int32 number = 1; } }', ',merge_oneof_messages=true')
if scalar.returncode or 'GoGoProtoPackageIsVersion12' in (third / 'sample.pb.go').read_text():
    raise RuntimeError('scalar-only file unexpectedly needs merge metadata')
print('oneof opt-in diagnostics and unchanged defaults passed')
