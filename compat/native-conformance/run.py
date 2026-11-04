#!/usr/bin/env python3
"""Compare native HYBRID with official Go against a pinned C++ test inventory."""
import argparse
import hashlib
import importlib.util
import json
import pathlib
import re
import subprocess

HERE = pathlib.Path(__file__).resolve().parent
spec = importlib.util.spec_from_file_location('observe', HERE.parent / 'conformance/run.py')
observer = importlib.util.module_from_spec(spec)
spec.loader.exec_module(observer)


def validate(names, result, baseline):
    tests, failures, skips, summaries, code = result
    return (code in (0, 1) and tests == set(names) and not skips
            and len(summaries) == 1 and set(failures) == set(baseline))


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('runner', type=pathlib.Path)
    parser.add_argument('testee', type=pathlib.Path)
    parser.add_argument('output', type=pathlib.Path)
    parser.add_argument('--official-testee', type=pathlib.Path, required=True)
    args = parser.parse_args()
    out = args.output.resolve()
    out.mkdir(parents=True)
    manifest = HERE / 'tests.json'
    inventory = json.loads(manifest.read_text())
    baseline = json.loads((HERE / 'official-failures.json').read_text())
    report = {'upstream': '36.2', 'profile': 'native HYBRID proto3 all-types',
              'inventory_sha256': hashlib.sha256(manifest.read_bytes()).hexdigest(),
              'baseline_sha256': hashlib.sha256((HERE / 'official-failures.json').read_bytes()).hexdigest(),
              'suites': {}, 'status': 'running'}
    errors = []
    report['source_revision'] = subprocess.check_output(['git', 'rev-parse', 'HEAD'], cwd=HERE.parents[1], text=True).strip()
    report['working_tree'] = subprocess.check_output(['git', 'status', '--porcelain'], cwd=HERE.parents[1], text=True).strip()
    report['build_info'] = {}
    versions = []
    for mode, executable in [('official', args.official_testee), ('native', args.testee)]:
        info = subprocess.check_output(['go', 'version', '-m', str(executable.resolve())], text=True)
        report['build_info'][mode] = info
        version = re.search(r'dep\s+google.golang.org/protobuf\s+(\S+)', info)
        if version is None:
            errors.append(mode + '-missing-runtime-version')
        else:
            versions.append(version.group(1))
    if len(set(versions)) != 1:
        errors.append('runtime-versions-differ')
    for suite, names in inventory.items():
        observed = {}
        for mode, executable in [('official', args.official_testee), ('native', args.testee)]:
            command = [str(args.runner.resolve()), suite, '--maximum_edition', '2023',
                       '--enforce_recommended', '--verbose', '--output_dir', str(out)]
            for name in names:
                command += ['--test', name]
            result = subprocess.run(command + [str(executable.resolve())], text=True,
                                    stdout=subprocess.PIPE, stderr=subprocess.STDOUT)
            (out / (suite + '-' + mode + '.log')).write_text(result.stdout)
            tests, failures, skips, summaries = observer.observe(result.stdout)
            observed[mode] = (tests, failures, skips, summaries, result.returncode)
            if not validate(names, observed[mode], baseline[suite]):
                errors.append(suite + '-' + mode)
            report['suites'][suite + '-' + mode] = {
                'tests': len(tests), 'failures': failures, 'skips': sorted(skips), 'summary': summaries}
        if observed['native'][1] != observed['official'][1]:
            errors.append(suite + '-official-difference')
        print(suite, len(names), 'tests;', len(observed['native'][1]),
              'shared official failures;', len(observed['native'][2]), 'skips', flush=True)
    report['status'] = 'failed' if errors else 'passed'
    report['gate_errors'] = errors
    (out / 'report.json').write_text(json.dumps(report, indent=2) + '\n')
    if errors:
        raise RuntimeError('Native conformance differs: ' + ','.join(errors))


if __name__ == '__main__':
    main()
