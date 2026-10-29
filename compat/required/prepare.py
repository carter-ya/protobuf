#!/usr/bin/env python3
"""Exercise required-field word boundaries with freshly generated consumers."""
import argparse
import pathlib
import shutil
import subprocess

ROOT = pathlib.Path(__file__).resolve().parents[2]
COUNTS = [63, 64, 65, 128, 129]


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('output', type=pathlib.Path)
    args = parser.parse_args()
    output = args.output.resolve()
    output.mkdir(parents=True, exist_ok=True)
    generator = output / 'gogo'
    subprocess.run(['go', 'build', '-o', str(generator), './protoc-gen-gogo'], cwd=ROOT, check=True)
    modern = subprocess.check_output(['protoc', '--version'], text=True).strip() == 'libprotoc 36.2'
    syntaxes = ['proto2', 'editions'] if modern else ['proto2']
    for syntax in syntaxes:
        for count in COUNTS:
            name = syntax + str(count)
            dest = output / name
            dest.mkdir()
            header = 'edition = "2023";' if syntax == 'editions' else 'syntax = "proto2";'
            fields = []
            for i in range(1, count + 1):
                number = 4096 if i == count else i
                fields.append(('int32 f%d = %d [features.field_presence=LEGACY_REQUIRED];' if syntax == 'editions'
                               else 'required int32 f%d = %d;') % (i, number))
            source = header + '\npackage ' + name + ';\noption go_package="example.com/gogo-required/' + name + '";\n'
            source += 'message Wide {\n' + '\n'.join(fields) + '\n}\nmessage Outer { optional Wide value=1; }\n'
            if syntax == 'editions':
                source = source.replace('optional Wide value', 'Wide value')
            path = output / (name + '.proto')
            path.write_text(source)
            subprocess.run(['protoc', '-I' + str(output), '--plugin=protoc-gen-test=' + str(generator),
                            '--test_out=paths=source_relative:' + str(dest), path.name], check=True)
            code = (dest / (name + '.pb.go')).read_text()
            version = 7 if count > 64 else 5 if syntax == 'editions' else 3
            if 'GoGoProtoPackageIsVersion%d' % version not in code:
                raise RuntimeError('wrong guard for ' + name)
    imports = '\n'.join('%s "example.com/gogo-required/%s"' % (syntax + str(n), syntax + str(n))
                        for syntax in syntaxes for n in COUNTS)
    cases = '\n'.join('{%d, new(%s.Wide), new(%s.Outer)},' % (n, syntax + str(n), syntax + str(n))
                      for syntax in syntaxes for n in COUNTS)
    template = (ROOT / 'compat/required/consumer_test.go.txt').read_text()
    (output / 'consumer_test.go').write_text(template.replace('IMPORTS', imports).replace('CASES', cases))
    (output / 'go.mod').write_text('module example.com/gogo-required\ngo 1.15\nrequire github.com/gogo/protobuf v1.3.2\nreplace github.com/gogo/protobuf => ' + str(ROOT) + '\n')
    shutil.copyfile(ROOT / 'go.sum', output / 'go.sum')
    subprocess.run(['go', 'test', './...'], cwd=output, check=True)


if __name__ == '__main__':
    main()
