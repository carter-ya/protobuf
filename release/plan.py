#!/usr/bin/env python3
"""Produce a reviewable component tag plan from completed remote reports; never publish."""
import argparse
import hashlib
import json
import pathlib
import re

from matrix import expected, PROFILE
from selection import check_frozen_identity, component_path, component_revisions, frozen_component, publish_components

HERE = pathlib.Path(__file__).resolve().parent
BACKENDS = {'gogo', 'gogofast', 'gogofaster', 'gogoslick', 'gogotypes'}


def qualify(reports, manifest, manifest_hash, revision):
    if manifest.get('qualification_profile') != PROFILE:
        raise ValueError('unsupported candidate qualification profile')
    observed = set()
    modes = {r['published_version_mode'] for r in reports}
    if len(modes) != 1:
        raise ValueError('commit-based and published-version reports must not be mixed')
    module_ids = {}
    origins = component_revisions(manifest, revision)
    for report in reports:
        if report['status'] != 'passed' or report['revision'] != revision or report['public_commit'] != revision:
            raise ValueError('reports must pass at the selected immutable public revision')
        if report['repository'] != manifest['repository'] or report['manifest_sha256'] != manifest_hash or not report['cold_module_cache']:
            raise ValueError('repository, contract or cold-cache provenance differs')
        if report.get('compiler') != 'libprotoc ' + manifest['compiler']:
            raise ValueError('unqualified compiler in remote evidence')
        minor = int(re.search(r'go1\.(\d+)', report['go']).group(1))
        if report['profile'] not in ['legacy', 'legacy-and-native']:
            raise ValueError('unknown consumer profile')
        legacy = report['profile'] == 'legacy'
        if minor not in [15, 23, 27] or minor == 15 and not legacy:
            raise ValueError('unsupported release qualification toolchain/profile')
        if report.get('qualification_profile') != PROFILE:
            raise ValueError('OPAQUE report profile missing')
        components = ['root'] if legacy else ['root', 'reflect']
        if report.get('component_revisions') != {name: origins[name] for name in components} or set(report['modules']) != set(components):
            raise ValueError('component origin evidence is missing or mixed')
        for name in components:
            if (report['published_version_mode'] or frozen_component(manifest, name)) and report.get('published_tag_commits', {}).get(name) != origins[name]:
                raise ValueError('published component tag origin was not verified')
        required_matrix = expected(legacy)
        runs = [(r['backend'], r['profile'], r['variant']) for r in report['consumer_runs']]
        if len(runs) != len(required_matrix) or set(runs) != required_matrix:
            raise ValueError('remote consumer matrix incomplete or duplicated')
        required_checks = {'go-version', 'compiler-version', 'download-root', 'install-generators'}
        if not legacy:
            required_checks |= {'download-reflect', 'download-runtime-pin', 'stable-override', 'stable-metadata-control'}
        if not required_checks <= {c['name'] for c in report['checks']}:
            raise ValueError('required provenance or control commands missing')
        for check in report['checks']:
            if check['returncode'] and not (check['name'] == 'stable-metadata-control' and check['returncode'] == 1 and check['expected_failure']):
                raise ValueError('a qualification command did not succeed')
        if not legacy:
            gap = report.get('stable_reference_control', {})
            if not gap.get('expected_diagnostic_reproduced') or gap.get('capability_gap') != manifest['official_runtime']['stable_reference_gap']['kind'] or gap.get('version') != manifest['official_runtime']['stable_reference']:
                raise ValueError('stable dependency decision lacks its exact control')
        for name, data in report['modules'].items():
            expected_path = component_path(manifest, name, manifest['repository'])
            if name not in ['root', 'reflect'] or data['Path'] != expected_path:
                raise ValueError('downloaded module path differs')
            if report['published_version_mode'] and data['Version'] != manifest[name]['version']:
                raise ValueError('published evidence did not install the manifest version')
            if (not legacy or data.get('Origin')) and data.get('Origin', {}).get('Hash') != origins[name]:
                raise ValueError('downloaded module origin differs')
            check_frozen_identity(manifest, name, data)
            identity = tuple(data[k] for k in ['Path', 'Version', 'Sum', 'GoModSum'])
            if name in module_ids and module_ids[name] != identity:
                raise ValueError('module versions/checksums differ across toolchains')
            module_ids[name] = identity
        observed.add((minor, legacy))
    if not {(15, True), (23, False), (27, False)} <= observed:
        raise ValueError('Go 1.15 legacy and Go 1.23/1.27 complete modern reports are required')
    if not {'root', 'reflect'} <= module_ids.keys():
        raise ValueError('paired modules were not both downloaded')
    return module_ids


def publication_plan(manifest, revision, reports, modules, report_paths):
    publishing = publish_components(manifest)
    tags = [manifest[name]['tag'] for name in publishing]
    published = all(r['published_version_mode'] for r in reports)
    plan = {'state': 'published-versions-verified' if published else 'prepared-not-published', 'revision': revision,
            'repository': manifest['repository'], 'tags': tags,
            'component_revisions': component_revisions(manifest, revision),
            'reused_components': {name: {'tag': manifest[name]['tag'], 'revision': component_revisions(manifest, revision)[name],
                'module': manifest['published_pair']['modules'][name]} for name in ['root', 'reflect'] if name not in publishing},
            'remote_module_identities': modules, 'reports': report_paths,
            'required_before_execution': ['exact-head clean review and all applicable CI successful',
                                          'selected revision is contained in ' + manifest['trunk'],
                                          'proposed remote tags absent or already identical; reused tags unchanged; never overwrite'],
            'commands_for_maintainer_review_only': [] if published else [
                'git tag -a ' + tag + ' ' + revision + ' -F release/' + ('STABLE-NOTES.md' if publishing == ['root'] else 'NOTES.md') for tag in tags] +
                ['git push --atomic origin ' + ' '.join('refs/tags/' + tag for tag in tags)],
            'required_after_tag_push': 'rerun release/remote.py --published for the selected revision and verify each component tag origin before announcing the release',
            'publication_performed': False}
    return plan


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--revision', required=True)
    parser.add_argument('--report', type=pathlib.Path, action='append', required=True)
    parser.add_argument('--output', type=pathlib.Path, required=True)
    args = parser.parse_args()
    if not re.fullmatch('[0-9a-f]{40}', args.revision):
        parser.error('use an exact commit SHA')
    raw = (HERE / 'manifest.json').read_bytes()
    manifest = json.loads(raw)
    reports = [json.loads(p.read_text()) for p in args.report]
    modules = qualify(reports, manifest, hashlib.sha256(raw).hexdigest(), args.revision)
    plan = publication_plan(manifest, args.revision, reports, modules, [str(p.resolve()) for p in args.report])
    args.output.write_text(json.dumps(plan, indent=2) + '\n')
    print('Component publication plan prepared; no tags or Releases were created.')


if __name__ == '__main__':
    main()
