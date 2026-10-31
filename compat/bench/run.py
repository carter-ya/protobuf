#!/usr/bin/env python3
"""Record reproducible codec/allocation/generated-size baselines; no speed gate."""
import argparse
import csv
import hashlib
import json
import os
import pathlib
import platform
import re
import statistics
import subprocess
import tarfile

ROOT = pathlib.Path(__file__).resolve().parents[2]
PRE_EXTENSION = '9cba925c0b5bf1a8ed530583d0af6c1afa1af019'
BACKENDS = ('gogo', 'gogofast', 'gogofaster', 'gogoslick', 'gogotypes')
PROFILES = {
    'legacy-v1.3.2': ('v1.3.2', 'proto2', False),
    'legacy-current': ('current', 'proto2', False),
    'optional-current': ('current', 'proto3', False),
    'edition2023-current': ('current', '2023', False),
    'edition2026-current': ('current', '2026', False),
    'extensions-pre13': (PRE_EXTENSION, '2023', True),
    'extensions-current': ('current', '2023', True),
}


def output(command, cwd=ROOT):
    return subprocess.check_output(command, cwd=cwd, text=True).strip()


def archive(revision, destination):
    destination.mkdir(parents=True)
    with subprocess.Popen(['git', 'archive', revision], cwd=ROOT,
                          stdout=subprocess.PIPE) as process:
        with tarfile.open(fileobj=process.stdout, mode='r|') as stream:
            stream.extractall(destination, filter='data')
        if process.wait():
            raise RuntimeError('unable to archive ' + revision)
    # Production runtime/generator packages are self-contained. Remove only the
    # archived snapshot's developer-tool dependencies, not production Go source.
    (destination / 'go.mod').write_text('module github.com/gogo/protobuf\n\ngo 1.15\n')
    return destination


def schema(syntax, extensions):
    header = ('syntax = "' + syntax + '";' if syntax.startswith('proto')
              else 'edition = "' + syntax + '";')
    optional = 'optional ' if syntax.startswith('proto') else ''
    packed = ' [packed = true]' if syntax == 'proto2' else ''
    extra = ('extensions 100 to 199;' if extensions else '')
    return header + '''
package benchmark;
option go_package = "example.com/gogo-benchmark/pb;pb";
message Child {
 OPTIONALint32 a = 1;
 OPTIONALint32 b = 2;
 OPTIONALbytes data = 3;
}
message Record {
 OPTIONALint32 counter = 1;
 OPTIONALstring label = 2;
 OPTIONALbytes data = 3;
 repeated uint32 samples = 4PACKED;
 Child child = 5;
 map<string, int32> states = 6;
 OPTIONAL_CHILDChild payload = 7;
 EXTRA
}
'''.replace('OPTIONAL_CHILD', 'optional ' if syntax == 'proto2' else '').replace('OPTIONAL', optional).replace('PACKED', packed).replace('EXTRA', extra).replace(
        ' Child child = 5;', ' ' + ('optional ' if syntax == 'proto2' else '') + 'Child child = 5;') + (
        '\nextend Record { Child extension_payload = 100; }\n' if extensions else '')


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('output', type=pathlib.Path)
    parser.add_argument('--backend', action='append', choices=BACKENDS)
    parser.add_argument('--profile', action='append', choices=PROFILES)
    parser.add_argument('--count', type=int, default=3)
    parser.add_argument('--benchtime', default='100ms')
    args = parser.parse_args()
    if args.count < 1:
        parser.error('--count must be positive')
    root = args.output.resolve()
    root.mkdir(parents=True, exist_ok=True)
    if (root / 'modules').exists():
        parser.error('use a fresh output directory; existing results are retained')
    backends, profiles = args.backend or BACKENDS, args.profile or list(PROFILES)
    environment = {
        'current_commit': output(['git', 'rev-parse', 'HEAD']),
        'working_tree_status': output(['git', 'status', '--porcelain']),
        'go': output(['go', 'version']), 'protoc': output(['protoc', '--version']),
        'platform': platform.platform(), 'architecture': platform.machine(),
        'cpu': platform.processor() or platform.machine(),
        'benchmark_script_sha256': hashlib.sha256(pathlib.Path(__file__).read_bytes()).hexdigest(),
        'benchmark_template_sha256': hashlib.sha256((ROOT / 'compat/bench/benchmark_test.go.txt').read_bytes()).hexdigest(),
        'count': args.count, 'benchtime': args.benchtime, 'GOMAXPROCS': 1,
        'legacy_revision': output(['git', 'rev-parse', 'v1.3.2']),
        'pre_extension_revision': PRE_EXTENSION,
        'production_only_snapshot_go_mod': True,
    }
    if platform.system() == 'Darwin':
        try:
            environment['cpu'] = subprocess.check_output(['sysctl', '-n', 'machdep.cpu.brand_string'], text=True, stderr=subprocess.DEVNULL).strip()
        except (OSError, subprocess.CalledProcessError):
            pass
    (root / 'environment.json').write_text(json.dumps(environment, indent=2) + '\n')
    sources = {'current': ROOT}
    for revision in sorted({PROFILES[p][0] for p in profiles} - {'current'}):
        sources[revision] = archive(revision, root / 'snapshots' / revision)
    drivers = {}
    for revision in sources:
        for backend in backends:
            driver = root / 'drivers' / revision / backend
            driver.parent.mkdir(parents=True, exist_ok=True)
            subprocess.run(['go', 'build', '-o', str(driver), './protoc-gen-' + backend],
                           cwd=sources[revision], check=True)
            drivers[revision, backend] = driver
    records = []
    for profile in profiles:
        revision, syntax, extensions = PROFILES[profile]
        for backend in backends:
            module = root / 'modules' / profile / backend
            package = module / 'pb'
            package.mkdir(parents=True)
            (module / 'record.proto').write_text(schema(syntax, extensions))
            subprocess.run(['protoc', '-I' + str(module), '--plugin=protoc-gen-test=' +
                            str(drivers[revision, backend]), '--test_out=paths=source_relative:' +
                            str(package), 'record.proto'], check=True)
            (module / 'go.mod').write_text('module example.com/gogo-benchmark\n\ngo 1.15\n'
                'require github.com/gogo/protobuf v1.3.2\nreplace github.com/gogo/protobuf => '
                + str(sources[revision]) + '\n')
            (module / 'go.sum').write_text((ROOT / 'go.sum').read_text())
            reader = ('func readExtension(m *pb.Record)(interface{},error) {return proto.GetExtension(m,pb.E_ExtensionPayload)}'
                      if extensions else 'func readExtension(m *pb.Record)(interface{},error) {return nil,nil}')
            test = (ROOT / 'compat/bench/benchmark_test.go.txt').read_text().replace(
                'HAS_EXTENSION', str(extensions).lower()).replace('READ_EXTENSION', reader)
            (module / 'benchmark_test.go').write_text(test)
            command = ['go', 'test', '-run', '^$', '-bench', '.', '-benchmem', '-cpu=1',
                       '-count=' + str(args.count), '-benchtime=' + args.benchtime]
            env = dict(os.environ, GOMAXPROCS='1')
            result = subprocess.run(command, cwd=module, env=env, text=True,
                                    stdout=subprocess.PIPE, stderr=subprocess.STDOUT)
            log = root / (profile + '-' + backend + '.log')
            log.write_text(result.stdout)
            cpu = re.search(r'^cpu: (.+)$', result.stdout, re.MULTILINE)
            if cpu:
                environment['cpu'] = cpu.group(1)
                (root / 'environment.json').write_text(json.dumps(environment, indent=2) + '\n')
            if result.returncode:
                raise RuntimeError('benchmark failed; see ' + str(log))
            metrics = {}
            for line in result.stdout.splitlines():
                if not line.startswith('Benchmark'):
                    continue
                name = re.sub(r'-[0-9]+$', '', line.split()[0])
                values = {unit: float(value) for value, unit in
                          re.findall(r'([0-9]+(?:\.[0-9]+)?(?:[eE][+-]?[0-9]+)?)\s+(ns/op|B/op|allocs/op|wire-B)', line)}
                if set(values) != {'ns/op', 'B/op', 'allocs/op', 'wire-B'}:
                    raise RuntimeError('unexpected benchmark row ' + line)
                metrics.setdefault(name, []).append(values)
            expected = 6 if extensions else 4
            if len(metrics) != expected or any(len(rows) != args.count for rows in metrics.values()):
                raise RuntimeError('benchmark result inventory changed: ' + str(log))
            generated = sum(path.stat().st_size for path in package.glob('*.pb.go'))
            for name, samples in metrics.items():
                row = {'profile': profile, 'backend': backend, 'operation': name,
                       'generated_bytes': generated, 'samples': args.count}
                row.update({unit: statistics.median(sample[unit] for sample in samples)
                            for unit in samples[0]})
                records.append(row)
            print(profile, backend, 'recorded', flush=True)
    (root / 'results.json').write_text(json.dumps(records, indent=2) + '\n')
    with (root / 'results.csv').open('w', newline='') as stream:
        writer = csv.DictWriter(stream, fieldnames=list(records[0]))
        writer.writeheader()
        writer.writerows(records)
    print(root / 'results.csv')


if __name__ == '__main__':
    main()
