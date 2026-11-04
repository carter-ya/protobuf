#!/usr/bin/env python3
"""Qualify a pinned preview separately from stable support and root dependencies."""
import argparse
import hashlib
import json
import os
import pathlib
import platform
import re
import shutil
import subprocess
import urllib.request
import zipfile

ROOT = pathlib.Path(__file__).resolve().parents[2]
RELEASE = 'v37.0-rc1'
GO_RELEASE = 'v1.37.0-rc.1'
ASSETS = {
    ('Linux', 'x86_64'): ('linux-x86_64', '9daf4de81ae85480013b49144f921e076c31cdc4589cfa0a40977dc9749df8f9'),
    ('Linux', 'aarch64'): ('linux-aarch_64', '78905909dfbcd75431bbc4a3ba584c77d7b0c9796512e7c994da8b40e1ffec07'),
    ('Darwin', 'arm64'): ('osx-aarch_64', 'd34cd8ecd4c0d9084e892227d63a2828eddaf1df498971cbfd18f9b00f6477f7'),
    ('Darwin', 'x86_64'): ('osx-x86_64', '9faf40c9280d69f557fe6ba7554882a71cd4211d4b00d620cbb02168e931897e'),
}
SCHEMAS = [
    'any.proto', 'api.proto', 'c_sharp_features.proto', 'compiler/plugin.proto',
    'cpp_features.proto', 'cpp_file_options.proto', 'cpp_options.proto',
    'descriptor.proto', 'duration.proto', 'empty.proto', 'field_mask.proto',
    'go_features.proto', 'java_features.proto', 'json_enumvalue_options.proto',
    'json_options.proto', 'source_context.proto', 'struct.proto', 'timestamp.proto',
    'type.proto', 'wrappers.proto',
]


def capture(command, cwd=ROOT):
    return subprocess.check_output([str(x) for x in command], cwd=cwd, text=True).strip()


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('output', type=pathlib.Path)
    parser.add_argument('--archive', type=pathlib.Path, help='reuse the official archive; SHA-256 is still checked')
    parser.add_argument('--stable-protoc', default=shutil.which('protoc'))
    parser.add_argument('--fuzztime', default='20s')
    args = parser.parse_args()
    if not args.stable_protoc or capture([args.stable_protoc, '--version']) != 'libprotoc 36.2':
        parser.error('the stable schema comparison requires protoc 36.2')
    stable = pathlib.Path(args.stable_protoc).resolve()
    go = capture(['go', 'version'])
    version = re.search(r'go1\.(\d+)', go)
    if version is None or int(version.group(1)) < 26:
        parser.error('Go 1.26+ is required by official Go Protobuf preview; root support stays Go 1.15')
    asset, digest = ASSETS.get((platform.system(), platform.machine()), (None, None))
    if asset is None:
        parser.error('no pinned preview asset for this platform')
    output = args.output.resolve()
    if output.exists():
        parser.error('use a new output directory; previous reports are retained')
    output.mkdir(parents=True)
    archive = output / ('protoc-37.0-rc-1-' + asset + '.zip')
    if args.archive:
        shutil.copyfile(args.archive, archive)
    else:
        url = 'https://github.com/protocolbuffers/protobuf/releases/download/' + RELEASE + '/' + archive.name
        with urllib.request.urlopen(url, timeout=120) as response:
            archive.write_bytes(response.read())
    if hashlib.sha256(archive.read_bytes()).hexdigest() != digest:
        raise SystemExit('preview archive SHA-256 mismatch')
    tool = output / 'protoc'
    with zipfile.ZipFile(archive) as stream:
        stream.extractall(tool)
    protoc = tool / 'bin/protoc'
    protoc.chmod(0o755)
    if capture([protoc, '--version']) != 'libprotoc 37.0-rc1':
        raise SystemExit('unexpected preview compiler version')
    env = dict(os.environ, PATH=str(protoc.parent) + os.pathsep + os.environ['PATH'], GOTOOLCHAIN='local')
    report = {'source_commit': capture(['git', 'rev-parse', 'HEAD']),
              'working_tree_status': capture(['git', 'status', '--porcelain']),
              'go': go, 'compiler': '37.0-rc1', 'official_go': GO_RELEASE,
              'compiler_asset': archive.name, 'compiler_sha256': digest,
              'harness_sha256': hashlib.sha256(pathlib.Path(__file__).read_bytes()).hexdigest(),
              'audit_sha256': hashlib.sha256((ROOT / 'compat/preview/schema_audit.go.txt').read_bytes()).hexdigest(),
              'status': 'running', 'checks': []}
    report_path = output / 'qualification.json'

    def record():
        report_path.write_text(json.dumps(report, indent=2) + '\n')

    def run(name, command, cwd=ROOT, extra=None):
        with (output / (name + '.log')).open('w') as log:
            process = subprocess.run([str(x) for x in command], cwd=cwd,
                                     env=dict(env, **(extra or {})), stdout=log, stderr=subprocess.STDOUT)
        report['checks'].append({'name': name, 'passed': process.returncode == 0})
        record()
        if process.returncode:
            raise RuntimeError('preview qualification failed: ' + name + '; inspect its log')
        print(name, 'passed', flush=True)

    record()
    try:
        for name, compiler in [('stable', stable), ('preview', protoc)]:
            includes = compiler.parent.parent / 'include/google/protobuf'
            if sorted(str(path.relative_to(includes)) for path in includes.rglob('*.proto')) != sorted(SCHEMAS):
                raise RuntimeError('official include file inventory changed')
            run('schema-' + name, [compiler, '-I' + str(compiler.parent.parent / 'include'),
                '--include_imports', '--descriptor_set_out=' + str(output / (name + '.pb'))]
                + ['google/protobuf/' + name for name in SCHEMAS])
        audit = output / 'audit'
        audit.mkdir()
        shutil.copyfile(ROOT / 'compat/preview/schema_audit.go.txt', audit / 'audit.go')
        (audit / 'go.mod').write_text('module example.com/gogo-preview-audit\n\ngo 1.26\n\nrequire google.golang.org/protobuf ' + GO_RELEASE + '\n')
        shutil.copyfile(ROOT / 'compat/preview/go.sum', audit / 'go.sum')
        run('schema-audit', ['go', 'run', '-mod=mod', 'audit.go', output / 'stable.pb', output / 'preview.pb', output / 'schema-audit.json'], audit)
        run('build-official-generator', ['go', 'build', '-mod=mod', '-o', output / 'protoc-gen-go', 'google.golang.org/protobuf/cmd/protoc-gen-go'], audit)
        run('official-enum-json', ['python3', ROOT / 'compat/go-enum-json/prepare.py', output / 'official-enum-json',
            '--official-generator', output / 'protoc-gen-go', '--official-module', GO_RELEASE])
        run('official-enum-prefix', ['python3', ROOT / 'compat/go-enum-prefix/prepare.py', output / 'official-enum-prefix',
            '--official-generator', output / 'protoc-gen-go', '--official-module', GO_RELEASE])
        run('native-hybrid', ['python3', ROOT / 'compat/hybrid/prepare.py', output / 'native-hybrid',
            '--runtime-version', GO_RELEASE, '--official-generator', output / 'protoc-gen-go'])
        run('native-hybrid-mixed', ['python3', ROOT / 'compat/hybrid/mixed.py', output / 'native-hybrid-mixed',
            '--runtime-version', GO_RELEASE])
        report['schema_audit'] = json.loads((output / 'schema-audit.json').read_text())
        report['schema_audit']['raw_sha256'] = {name: hashlib.sha256((output / (name + '.pb')).read_bytes()).hexdigest() for name in ['stable', 'preview']}
        record()
        for name in ['check.sh', 'check-entrypoints.sh', 'check-optional-mappings.sh', 'check-oneof-merge.sh', 'check-compare-floats.sh', 'check-required.sh', 'check-required-merge.sh', 'check-legacy-enums.sh', 'check-go-enum-json.sh', 'check-go-enum-prefix.sh', 'check-go-optimize.sh', 'check-feature-inventory.sh']:
            run(name.removesuffix('.sh'), ['sh', ROOT / 'compat' / name])
        run('editions-all-backends', ['python3', ROOT / 'compat/fast/prepare.py', output / 'editions'])

        for source, module in [('qualify', 'differential'), ('oracle', 'oracle'), ('modern', 'modern')]:
            dest = output / module
            dest.mkdir()
            for path in (ROOT / 'compat' / source).glob('*.go'):
                shutil.copyfile(path, dest / path.name)
            corpus = ROOT / 'compat' / source / 'testdata'
            if corpus.exists():
                shutil.copytree(corpus, dest / 'testdata')
            text = (ROOT / 'compat' / source / 'go.mod').read_text()
            text = re.sub(r'^go .+$', 'go 1.26', text, flags=re.MULTILINE)
            text = re.sub(r'google.golang.org/protobuf v\S+', 'google.golang.org/protobuf ' + GO_RELEASE, text)
            text = re.sub(r'replace\s+github.com/gogo/protobuf\s+=>[^\n]+',
                          'replace github.com/gogo/protobuf => ' + str(ROOT), text)
            (dest / 'go.mod').write_text(text)
            original_sum = ROOT / 'compat' / source / 'go.sum'
            (dest / 'go.sum').write_text(original_sum.read_text() + (ROOT / 'compat/preview/go.sum').read_text())
            descriptor = dest / 'schema.pb'
            inputs = ['compat/editions/edition2026/sample.proto']
            if module == 'modern':
                inputs += ['compat/optional/reflection/sample.proto', 'compat/editions/edition2023/sample.proto',
                           'compat/editions/edition2023/foreign_ext.proto', 'compat/editions/edition2024/sample.proto',
                           'google/protobuf/any.proto', 'google/protobuf/wrappers.proto', 'google/protobuf/duration.proto']
            run(module + '-descriptor', [protoc, '-I' + str(ROOT), '-I' + str(ROOT / 'protobuf'),
                '--include_imports', '--descriptor_set_out=' + str(descriptor)] + inputs)
            variables = {'GOGO_ORACLE_DESCRIPTOR': str(descriptor), 'GOGO_MODERN_DESCRIPTOR': str(descriptor)}
            for flavor, flags in [('default', []), ('purego', ['-tags=purego']), ('race', ['-race'])]:
                run(module + '-' + flavor, ['go', 'test', '-mod=mod'] + flags + ['./...'], dest, variables)
            if module == 'differential':
                for target in ['FuzzText', 'FuzzJSON']:
                    run(target, ['go', 'test', '-mod=readonly', '-run=^$', '-fuzz=^' + target + '$',
                                 '-fuzztime=' + args.fuzztime, '-parallel=2'], dest, {'GOMAXPROCS': '2'})
        report['status'] = 'passed'
    except Exception:
        report['status'] = 'failed'
        raise
    finally:
        record()
    print(report_path)


if __name__ == '__main__':
    main()
