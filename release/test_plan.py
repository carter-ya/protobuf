import copy
import hashlib
import importlib.util
import json
import pathlib
import unittest
from selection import component_revisions, frozen_component

HERE = pathlib.Path(__file__).resolve().parent
spec = importlib.util.spec_from_file_location('plan', HERE / 'plan.py')
plan = importlib.util.module_from_spec(spec)
spec.loader.exec_module(plan)


class PublicationPlanGuards(unittest.TestCase):
    def setUp(self):
        raw = (HERE / 'manifest.json').read_bytes()
        self.manifest = json.loads(raw)
        self.digest = hashlib.sha256(raw).hexdigest()
        self.revision = '0123456789abcdef0123456789abcdef01234567'
        self.reports = []
        # In-memory unit fixtures only, never release evidence.
        for minor in [15, 23, 27]:
            legacy = minor == 15
            flavors = ['legacy'] if legacy else ['legacy', 'open', 'hybrid', 'opaque', 'edition2026']
            variants = ['default', 'purego'] if legacy else ['default', 'purego', 'race']
            checks = ['go-version', 'compiler-version', 'download-root', 'install-generators']
            if not legacy:
                checks += ['download-reflect', 'download-runtime-pin', 'stable-override', 'stable-metadata-control']
            report = {'status': 'passed', 'revision': self.revision, 'public_commit': self.revision,
                      'repository': self.manifest['repository'], 'manifest_sha256': self.digest,
                      'qualification_profile': 'native-opaque-v1', 'cold_module_cache': True, 'published_version_mode': False,
                      'go': 'go version go1.' + str(minor) + '.0 linux/amd64',
                      'profile': 'legacy' if legacy else 'legacy-and-native',
                      'compiler': 'libprotoc ' + self.manifest['compiler'],
                      'consumer_runs': [{'backend': b, 'profile': f, 'variant': v}
                                        for b in plan.BACKENDS for f in flavors for v in variants + ([] if legacy or f not in ['hybrid','edition2026'] else ['protoopaque','protoopaque-purego','protoopaque-race'])],
                      'checks': [{'name': n, 'returncode': int(n == 'stable-metadata-control'),
                                  'expected_failure': n == 'stable-metadata-control'} for n in checks],
                      'modules': {name: {'Path': self.manifest['repository'] + ('/reflect' if name == 'reflect' else ''),
                                         'Version': 'v0.0.0-20261101135246-0123456789ab',
                                         'Sum': 'h1:unit-only', 'GoModSum': 'h1:unit-only',
                                         'Origin': {'Hash': self.revision}}
                                  for name in ['root'] if legacy} if legacy else {},
                      'stable_reference_control': {'version': self.manifest['official_runtime']['stable_reference'],
                          'expected_diagnostic_reproduced': True,
                          'capability_gap': self.manifest['official_runtime']['stable_reference_gap']['kind']}}
            if not legacy:
                report['modules'] = copy.deepcopy(self.reports[0]['modules'])
                report['modules']['reflect'] = copy.deepcopy(report['modules']['root'])
                report['modules']['reflect']['Path'] += '/reflect'
                if frozen_component(self.manifest, 'reflect'):
                    report['modules']['reflect'] = copy.deepcopy(self.manifest['published_pair']['modules']['reflect'])
                    report['modules']['reflect']['Origin'] = {'Hash': component_revisions(self.manifest, self.revision)['reflect']}
            components = ['root'] if legacy else ['root', 'reflect']
            report['component_revisions'] = {name: component_revisions(self.manifest, self.revision)[name] for name in components}
            report['published_tag_commits'] = {name: report['component_revisions'][name] for name in components if frozen_component(self.manifest, name)}
            self.reports.append(report)

    def test_rejects_unqualified_publication_evidence(self):
        plan.qualify(self.reports, self.manifest, self.digest, self.revision)
        for kind in ['revision', 'failed', 'manifest', 'matrix', 'duplicate', 'checksum', 'missing-control', 'missing-command', 'mode', 'control-crash', 'missing-opaque', 'wrong-compiler', 'missing-alternate', 'missing-go115', 'component-origin', 'missing-reflect', 'reused-checksum', 'reused-origin']:
            bad = copy.deepcopy(self.reports)
            if kind == 'revision':
                bad[1]['revision'] = 'f' * 40
            elif kind == 'failed':
                bad[1]['status'] = 'failed'
            elif kind == 'manifest':
                bad[1]['manifest_sha256'] = 'bad'
            elif kind == 'matrix':
                bad[1]['consumer_runs'].pop()
            elif kind == 'duplicate':
                bad[1]['consumer_runs'].append(copy.deepcopy(bad[1]['consumer_runs'][0]))
            elif kind == 'checksum':
                bad[1]['modules']['root']['Sum'] = 'h1:different'
            elif kind == 'missing-control':
                bad[1]['stable_reference_control'] = {}
            elif kind == 'missing-command':
                bad[1]['checks'].pop()
            elif kind == 'mode':
                bad[1]['published_version_mode'] = True
            elif kind == 'control-crash':
                next(c for c in bad[1]['checks'] if c['name'] == 'stable-metadata-control')['returncode'] = 137
            elif kind == 'wrong-compiler':
                bad[1]['compiler'] = 'libprotoc 21.8'
            elif kind == 'missing-opaque':
                bad[1]['consumer_runs'] = [r for r in bad[1]['consumer_runs'] if r['profile'] != 'opaque']
            elif kind == 'missing-alternate':
                bad[1]['consumer_runs'] = [r for r in bad[1]['consumer_runs'] if not r['variant'].startswith('protoopaque')]
            elif kind == 'component-origin':
                bad[1]['component_revisions']['reflect'] = self.revision
            elif kind == 'missing-reflect':
                bad[1]['modules'].pop('reflect')
            elif kind == 'reused-checksum':
                bad[1]['modules']['reflect']['Sum'] = 'h1:changed'
            elif kind == 'reused-origin':
                bad[1]['published_tag_commits']['reflect'] = self.revision
            else:
                bad.pop(0)
            with self.subTest(kind=kind), self.assertRaises(ValueError):
                plan.qualify(bad, self.manifest, self.digest, self.revision)

    def test_root_only_plan_and_published_verification(self):
        modules = plan.qualify(self.reports, self.manifest, self.digest, self.revision)
        result = plan.publication_plan(self.manifest, self.revision, self.reports, modules, [])
        self.assertEqual(result['tags'], [self.manifest['root']['tag']])
        self.assertEqual(set(result['reused_components']), {'reflect'})
        self.assertEqual(result['reused_components']['reflect']['revision'], self.manifest['published_pair']['revisions']['reflect'])
        self.assertEqual(result['commands_for_maintainer_review_only'], [
            'git tag -a ' + self.manifest['root']['tag'] + ' ' + self.revision + ' -F release/STABLE-NOTES.md',
            'git push --atomic origin refs/tags/' + self.manifest['root']['tag']])
        published = copy.deepcopy(self.reports)
        for report in published:
            report['published_version_mode'] = True
            for name in report['modules']:
                report['modules'][name]['Version'] = self.manifest[name]['version']
                report['published_tag_commits'][name] = report['component_revisions'][name]
        modules = plan.qualify(published, self.manifest, self.digest, self.revision)
        self.assertEqual(plan.publication_plan(self.manifest, self.revision, published, modules, [])['commands_for_maintainer_review_only'], [])
        published[1]['published_tag_commits']['reflect'] = self.revision
        with self.assertRaises(ValueError):
            plan.qualify(published, self.manifest, self.digest, self.revision)

    def test_paired_rc_mode_still_qualifies_both_components(self):
        manifest, reports = copy.deepcopy(self.manifest), copy.deepcopy(self.reports)
        manifest['state'] = 'release-candidate'
        manifest['root']['version'] = manifest['root']['tag'] = 'v1.4.0-rc.3'
        manifest['reflect'].update(version='v0.1.0-rc.3', tag='reflect/v0.1.0-rc.3', minimum_root='v1.4.0-rc.3')
        for report in reports:
            report['published_tag_commits'] = {}
            for name in report['modules']:
                report['component_revisions'][name] = self.revision
                report['modules'][name]['Origin'] = {'Hash': self.revision}
                report['modules'][name]['Version'] = 'v0.0.0-20261101135246-0123456789ab'
        modules = plan.qualify(reports, manifest, self.digest, self.revision)
        result = plan.publication_plan(manifest, self.revision, reports, modules, [])
        self.assertEqual(result['tags'], ['v1.4.0-rc.3', 'reflect/v0.1.0-rc.3'])
        self.assertEqual(result['reused_components'], {})
        reports[1]['modules']['root'].pop('Origin')
        with self.assertRaises(ValueError):
            plan.qualify(reports, manifest, self.digest, self.revision)


if __name__ == '__main__':
    unittest.main()
