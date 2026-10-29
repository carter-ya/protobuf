#!/usr/bin/env python3
import argparse
import pathlib
import shutil
import subprocess

ROOT = pathlib.Path(__file__).resolve().parents[2]
BACKENDS = ['gogo', 'gogofast', 'gogofaster', 'gogoslick', 'gogotypes']


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('output', type=pathlib.Path)
    parser.add_argument('--schema', action='append')
    parser.add_argument('--backend', action='append')
    parser.add_argument('--edition', action='append')
    args = parser.parse_args()
    output = args.output.resolve()
    output.mkdir(parents=True)
    for backend in sorted(set(x.removesuffix("-aux") for x in (args.backend or BACKENDS))):
        subprocess.run(['go', 'build', '-o', str(output / backend), './protoc-gen-' + backend], cwd=ROOT, check=True)
    for schema in (args.schema or ['scalars', 'complex']):
        source = (ROOT / ('compat/fast/' + schema + '.proto')).read_text()
        for edition in (args.edition or ['2023', '2024', '2026']):
            for backend in (args.backend or BACKENDS + (["gogoslick-aux"] if schema == "complex" else [])):
                driver = backend.removesuffix("-aux")
                dest = output / (schema + '-' + edition + '-' + backend)
                dest.mkdir()
                (dest / 'pb').mkdir()
                text = source.replace('"2023"', '"' + edition + '"')
                if schema == 'complex' and edition == '2026':
                    text = text.replace('message Sample {', 'export message Sample {').replace('enum Closed {', 'export enum Closed {')
                (dest / (schema + '.proto')).write_text(text)
                inputs = [schema + '.proto']
                if schema == 'complex':
                    for extra in ['imported', 'extensions']:
                        text = (ROOT / ('compat/fast/' + extra + '.proto')).read_text().replace('"2023"', '"' + edition + '"')
                        (dest / (extra + '.proto')).write_text(text)
                        inputs.append(extra + '.proto')
                if backend.endswith('-aux'):
                    shutil.copytree(ROOT / 'gogoproto', dest / 'gogoproto')
                    for filename in inputs:
                        path = dest / filename
                        path.write_text(path.read_text() + '\nimport "gogoproto/gogo.proto";\n' + '\n'.join('option (gogoproto.' + opt + '_all) = true;' for opt in ['gostring','populate','equal','compare']))
                subprocess.run(['protoc', '-I' + str(dest), '-I' + str(ROOT), '--plugin=protoc-gen-test=' + str(output / driver), '--test_out=paths=source_relative:' + str(dest / 'pb')] + inputs, check=True)
                code = (dest / ('pb/' + schema + '.pb.go')).read_text()
                if backend != 'gogo' and 'GoGoProtoPackageIsVersion9' not in code:
                    raise RuntimeError('fast runtime guard missing')
                (dest / 'go.mod').write_text('module example.com/gogo-' + ('fast' if schema == 'scalars' else 'complex') + '\ngo 1.15\nrequire github.com/gogo/protobuf v1.3.2\nreplace github.com/gogo/protobuf => ' + str(ROOT) + '\n')
                shutil.copyfile(ROOT / 'go.sum', dest / 'go.sum')
                test = (ROOT / ('compat/fast/' + schema + '_test.go.txt')).read_text().replace('FAST', 'true' if backend != 'gogo' else 'false').replace('AUX', 'true' if backend.endswith('-aux') else 'false')
                (dest / (schema + '_test.go')).write_text(test)
                if backend.endswith('-aux'):
                    shutil.copyfile(ROOT / 'compat/fast/aux_test.go.txt', dest / 'aux_test.go')
                subprocess.run(['go', 'test', './...'], cwd=dest, check=True)
    if (output / 'gogo').exists():
        negative = output / 'negative'
        negative.mkdir()
        shutil.copyfile(ROOT / 'compat/fast/imported.proto', negative / 'imported.proto')
        cases = [
            ('scalars', '\noption (gogoproto.marshaler_all) = true;\n', 'marshaler requires sizer'),
            ('complex', '\noption (gogoproto.marshaler_all) = true;\noption (gogoproto.sizer_all) = true;\noption (gogoproto.unmarshaler_all) = true;\n', 'dependency requires matching codec methods'),
        ]
        for schema, options, expected in cases:
            text = (ROOT / ('compat/fast/' + schema + '.proto')).read_text()
            if schema == 'complex':
                text = text.replace('message Child {', 'message Child { option (gogoproto.unmarshaler) = false;')
            text += '\nimport "gogoproto/gogo.proto";\n' + options
            (negative / (schema + '.proto')).write_text(text)
            result = subprocess.run(['protoc', '-I' + str(negative), '-I' + str(ROOT),
                                     '--plugin=protoc-gen-test=' + str(output / 'gogo'),
                                     '--test_out=' + str(negative), schema + '.proto'],
                                    stdout=subprocess.PIPE, stderr=subprocess.STDOUT, text=True)
            if result.returncode == 0 or expected not in result.stdout:
                raise RuntimeError('partial fast configuration not diagnosed: ' + result.stdout)



if __name__ == '__main__':
    main()
