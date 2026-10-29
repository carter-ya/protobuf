#!/usr/bin/env python3
"""Gate observed upstream results against reviewed, exact-name baseline records."""
import argparse
import collections
import hashlib
import json
import pathlib
import re
import subprocess
import sys

ROOT = pathlib.Path(__file__).resolve().parent


def observe(log):
    tests = set(re.findall(r'^conformance test: name=([^,]+), request=', log, re.M))
    failures = {m[0]: m[1].split(', request=', 1)[0] for m in
                re.findall(r'^ERROR, test=([^:]+): (.*)$', log, re.M)}
    skips = set(re.findall(r'^SKIPPED, test=([^ ]+) ', log, re.M))
    summaries = re.findall(r'^CONFORMANCE SUITE (?:PASSED|FAILED):.*$', log, re.M)
    return tests, failures, skips, summaries


def check_results(suite, tests, failures, skips, summaries, returncode, expected):
    errors = []
    stable = {x['test'] for x in expected['failures']}
    flaky = {x['test'] for x in expected['variable_outcomes']}
    allowed_skips = {x['test'] for x in expected['skips']}
    inventory = hashlib.sha256('\n'.join(sorted(tests)).encode()).hexdigest()
    if len(summaries) != 1 or returncode not in (0, 1):
        errors.append(suite + ': runner failed or did not finish')
    if inventory != expected['inventory_sha256'] or len(tests) != expected['tests']:
        errors.append(suite + ': test inventory changed')
    for name in sorted(set(failures) - stable - flaky):
        errors.append(suite + ': new failure: ' + name)
    for name in sorted(stable - set(failures)):
        errors.append(suite + ': baseline failure now passes; remove its record: ' + name)
    if skips != allowed_skips:
        errors.append(suite + ': skipped tests changed: ' + repr(sorted(skips ^ allowed_skips)))
    if not (stable | flaky | allowed_skips) <= tests:
        errors.append(suite + ': baseline contains a nonexistent test')
    return errors


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('runner', type=pathlib.Path)
    parser.add_argument('testee', type=pathlib.Path)
    parser.add_argument('output', type=pathlib.Path)
    parser.add_argument('--baseline', type=pathlib.Path, default=ROOT / 'failures.json')
    args = parser.parse_args()
    if not args.runner.is_file() or not args.testee.is_file():
        sys.exit('conformance runner and testee are required; tests cannot be skipped')
    baseline = json.loads(args.baseline.read_text())
    output = args.output.resolve()
    output.mkdir(parents=True, exist_ok=True)
    report = {'upstream': baseline['upstream'], 'mode': 'legacy', 'suites': {}}
    errors = []
    for suite in ['binary_json', 'text']:
        dest = output / suite
        dest.mkdir(exist_ok=True)
        process = subprocess.run([str(args.runner.resolve()), suite, '--maximum_edition', '2023',
                                  '--enforce_recommended', '--verbose', '--output_dir', str(dest),
                                  str(args.testee.resolve())], stdout=subprocess.PIPE, stderr=subprocess.STDOUT, text=True)
        (dest / 'runner.log').write_text(process.stdout)
        tests, failures, skips, summaries = observe(process.stdout)
        expected = baseline[suite]
        flaky = {x['test'] for x in expected['variable_outcomes']}
        errors.extend(check_results(suite, tests, failures, skips, summaries, process.returncode, expected))
        counts = collections.Counter('.'.join(name.split('.')[:2]) for name in failures)
        report['suites'][suite] = {'tests': len(tests), 'passed': len(tests - set(failures) - skips),
                                  'failures': len(failures), 'skipped': len(skips),
                                  'failures_by_level_and_syntax': dict(sorted(counts.items())),
                                  'variable_failures': sorted(set(failures) & flaky),
                                  'summary': summaries, 'failures_detail': failures}
        print(suite + ': ' + ', '.join('%s=%s' % (k, report['suites'][suite][k]) for k in ['tests', 'passed', 'failures', 'skipped']))
    report['gate_errors'] = errors
    (output / 'report.json').write_text(json.dumps(report, indent=2, sort_keys=True) + '\n')
    if errors:
        sys.exit('\n'.join(errors))
    print('No new failures or skips. Known limitations remain recorded individually.')


if __name__ == '__main__':
    main()
