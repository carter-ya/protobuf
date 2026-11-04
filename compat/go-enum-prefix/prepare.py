#!/usr/bin/env python3
"""Qualify prefix stripping, migration aliases and imported default references."""
import argparse
import json
import pathlib
import re
import subprocess

ROOT = pathlib.Path(__file__).resolve().parents[2]
HERE = pathlib.Path(__file__).resolve().parent
BACKENDS = ['gogo', 'gogofast', 'gogofaster', 'gogoslick', 'gogotypes']
MODES = ['default', 'UNSPECIFIED', 'KEEP', 'GENERATE_BOTH', 'STRIP']


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('output', type=pathlib.Path)
    parser.add_argument('--official-generator', type=pathlib.Path)
    parser.add_argument('--official-module', default='v1.36.12')
    args = parser.parse_args()
    compiler = subprocess.check_output(['protoc', '--version'], text=True).strip()
    if compiler not in ['libprotoc 36.2', 'libprotoc 37.0-rc1']:
        parser.error('requires a qualified Editions compiler')
    out = args.output.resolve()
    out.mkdir(parents=True)
    backends = ['official'] if args.official_generator else BACKENDS
    editions = ['2024'] + (['2026'] if not args.official_generator or args.official_module != 'v1.36.12' else [])
    checks = []
    for backend in backends:
        driver = args.official_generator.resolve() if args.official_generator else out / backend
        if not args.official_generator:
            subprocess.run(['go', 'build', '-o', driver, './protoc-gen-' + backend], cwd=ROOT, check=True)
        for edition in editions:
            for mode in MODES:
                dest = out / (backend + '-' + edition + '-' + mode)
                dest.mkdir()
                for name in ['enums.proto', 'legacy.proto', 'bridge.proto', 'use.proto']:
                    source = (HERE / name).read_text().replace('"2024"', '"' + edition + '"')
                    source = source.replace('// FILE_PREFIX', '' if mode == 'default' else 'option features.(pb.go).strip_enum_prefix = STRIP_ENUM_PREFIX_' + mode + ';')
                    if edition == '2026' and name == 'enums.proto':
                        source = source.replace('option features.enum_type = CLOSED;', 'option features.enum_type = CLOSED;\noption features.default_symbol_visibility = EXPORT_ALL;')
                    (dest / name).write_text(source)
                for names, folder in [(['enums.proto'], 'pb'), (['legacy.proto'], 'pb'), (['bridge.proto'], 'bridge'), (['use.proto'], 'use')]:
                    (dest / folder).mkdir(exist_ok=True)
                    subprocess.run(['protoc', '-I' + str(dest), '--plugin=protoc-gen-test=' + str(driver),
                        '--test_out=paths=source_relative:' + str(dest / folder)] + names, check=True)
                filemode = 'KEEP' if mode in ['default', 'UNSPECIFIED'] else mode
                expected, absent = set(), set()
                def entry(prefix, original, short, selected=filemode):
                    old, new = prefix + original, prefix + short
                    if selected in ['KEEP', 'UNSPECIFIED']:
                        expected.add(old)
                        if old != new:
                            absent.add(new)
                        return old
                    expected.add(new)
                    if selected == 'GENERATE_BOTH':
                        expected.add(old)
                    elif old != new:
                        absent.add(old)
                    return new
                color_prefix = '' if backend == 'gogoslick' else 'Color_'
                nested_prefix = '' if backend == 'gogoslick' else 'Container_'
                mixed_prefix = '' if backend == 'gogoslick' else 'MixedCase_'
                kept_prefix = '' if backend == 'gogoslick' else 'Kept_'
                entry(color_prefix, 'COLOR_UNSPECIFIED', 'UNSPECIFIED')
                red = entry(color_prefix, 'COLOR_RED', 'RED')
                entry(color_prefix, 'COLOR_RED_ALIAS', 'RED_ALIAS')
                entry(color_prefix, 'COLOR_GREEN', 'GREEN', 'KEEP')
                entry(color_prefix, 'COLOR_BLUE', 'BLUE', 'STRIP')
                entry(color_prefix, 'COLOR_PURPLE', 'PURPLE', 'GENERATE_BOTH')
                entry(color_prefix, 'COLOR_YELLOW', 'YELLOW', 'UNSPECIFIED')
                entry(nested_prefix, 'DIRECTION_ORIGIN', 'ORIGIN')
                direction = entry(nested_prefix, 'DIRECTION_UP', 'UP')
                entry(mixed_prefix, 'M_i_X_E_D_C_A_S_E_ZERO', 'ZERO')
                mixed = entry(mixed_prefix, 'M_i_X_E_D_C_A_S_E_ONE', 'ONE')
                entry(mixed_prefix, 'MIXEDCASE', 'MIXEDCASE')
                entry(mixed_prefix, 'OTHER', 'OTHER')
                entry(kept_prefix, 'KEPT_DEFAULT', 'DEFAULT', 'KEEP')
                kept = entry(kept_prefix, 'KEPT_VALUE', 'VALUE', 'KEEP')
                generated = (dest / 'pb/enums.pb.go').read_text()
                declarations = set(re.findall(r'^\s*([A-Za-z_][A-Za-z_0-9]*)\s+[^\n=]+=', generated, re.MULTILINE))
                if not expected <= declarations or (absent - expected) & declarations:
                    raise RuntimeError('wrong public enum API: ' + repr((expected - declarations, (absent - expected) & declarations)))
                replacements = {'COLOR_RED_NAME': red, 'KEPT_VALUE_NAME': kept, 'DIRECTION_UP_NAME': direction, 'MIXED_ONE_NAME': mixed, 'PURPLE_OLD_NAME': color_prefix + 'COLOR_PURPLE', 'PURPLE_NEW_NAME': color_prefix + 'PURPLE'}
                for template in ['consumer_test.go.txt'] + ([] if args.official_generator else ['codec_test.go.txt']):
                    text = (HERE / template).read_text()
                    for old, new in replacements.items():
                        text = text.replace(old, new)
                    (dest / template.removesuffix('.txt')).write_text(text)
                if args.official_generator:
                    module = ('go 1.23\n' if args.official_module == 'v1.36.12' else 'go 1.26\n') + 'require google.golang.org/protobuf ' + args.official_module + '\n'
                    sums = (ROOT / 'compat/qualify/go.sum').read_text() + (ROOT / 'compat/preview/go.sum').read_text()
                else:
                    module = 'go 1.15\nrequire github.com/gogo/protobuf v1.3.2\nreplace github.com/gogo/protobuf => ' + str(ROOT) + '\n'
                    sums = (ROOT / 'go.sum').read_text()
                (dest / 'go.mod').write_text('module example.com/enum-prefix\n' + module)
                (dest / 'go.sum').write_text(sums)
                for variant, flags in [('default', []), ('purego', ['-tags=purego'])]:
                    subprocess.run(['go', 'test', '-count=1'] + flags + ['./...'], cwd=dest, check=True)
                    checks.append({'backend': backend, 'edition': edition, 'mode': mode, 'variant': variant})
    if not args.official_generator:
        negative = out / 'negative'
        negative.mkdir()
        for backend in BACKENDS:
            collision = 'RED' if backend == 'gogoslick' else 'Color_RED'
            source = ('edition="2024";package bad;import option "google/protobuf/go_features.proto";'
                'option features.(pb.go).strip_enum_prefix=STRIP_ENUM_PREFIX_STRIP;option features.enforce_naming_style=STYLE_LEGACY;'
                'enum Color {COLOR_ZERO=0;COLOR_RED=1;}message ' + collision + '{}\n')
            (negative / 'bad.proto').write_text(source)
            process = subprocess.run(['protoc', '-I' + str(negative), '--plugin=protoc-gen-test=' + str(out / backend),
                '--test_out=' + str(negative), 'bad.proto'], stdout=subprocess.PIPE, stderr=subprocess.STDOUT, text=True)
            (negative / (backend + '.log')).write_text(process.stdout)
            if process.returncode == 0 or 'stripped enum Go name' not in process.stdout:
                raise RuntimeError('stripped API collision not diagnosed: ' + process.stdout)
            # The same collision must be found across files in one Go package.
            (negative / 'bad.proto').write_text(source.replace('message ' + collision + '{}', ''))
            (negative / 'cross.proto').write_text('syntax="proto2";package bad;message ' + collision + '{}\n')
            process = subprocess.run(['protoc', '-I' + str(negative), '--plugin=protoc-gen-test=' + str(out / backend),
                '--test_out=' + str(negative), 'bad.proto', 'cross.proto'], stdout=subprocess.PIPE, stderr=subprocess.STDOUT, text=True)
            (negative / (backend + '-cross.log')).write_text(process.stdout)
            if process.returncode == 0 or 'stripped enum Go name' not in process.stdout:
                raise RuntimeError('cross-file stripped API collision not diagnosed: ' + process.stdout)
            custom = source.replace('enum Color {COLOR_ZERO=0;', 'import option "gogoproto/gogo.proto";enum Color {COLOR_ZERO=0 [(gogoproto.enumvalue_customname)="Renamed"];')
            custom = custom.replace('message ' + collision + '{}', '')
            (negative / 'bad.proto').write_text(custom)
            process = subprocess.run(['protoc', '-I' + str(negative), '-I' + str(ROOT), '-I' + str(ROOT / 'protobuf'),
                '--plugin=protoc-gen-test=' + str(out / backend), '--test_out=' + str(negative), 'bad.proto'],
                stdout=subprocess.PIPE, stderr=subprocess.STDOUT, text=True)
            (negative / (backend + '-custom.log')).write_text(process.stdout)
            if process.returncode == 0 or 'conflicts with a custom Go enum name' not in process.stdout:
                raise RuntimeError('ambiguous custom enum naming was not diagnosed: ' + process.stdout)
            keyword = ('edition="2024";package bad;import option "google/protobuf/go_features.proto";import option "gogoproto/gogo.proto";'
                'option features.enforce_naming_style=STYLE_LEGACY;option features.(pb.go).strip_enum_prefix=STRIP_ENUM_PREFIX_STRIP;'
                'option (gogoproto.goproto_enum_prefix_all)=false;enum Color {COLOR_type=0;}')
            (negative / 'bad.proto').write_text(keyword)
            process = subprocess.run(['protoc', '-I' + str(negative), '-I' + str(ROOT), '-I' + str(ROOT / 'protobuf'),
                '--plugin=protoc-gen-test=' + str(out / backend), '--test_out=' + str(negative), 'bad.proto'],
                stdout=subprocess.PIPE, stderr=subprocess.STDOUT, text=True)
            (negative / (backend + '-keyword.log')).write_text(process.stdout)
            if process.returncode == 0 or 'not a valid identifier' not in process.stdout:
                raise RuntimeError('invalid stripped Go identifier was not diagnosed: ' + process.stdout)


    (out / 'qualification.json').write_text(json.dumps({'compiler': compiler, 'checks': checks, 'negative_checks': 0 if args.official_generator else 20, 'status': 'passed'}, indent=2) + '\n')
    print('Enum prefix qualification passed:', len(checks), 'consumer runs')


if __name__ == '__main__':
    main()
