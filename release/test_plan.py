import copy
import hashlib
import importlib.util
import json
import pathlib
import unittest

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
            flavors = ['legacy'] if legacy else ['legacy', 'open', 'hybrid', 'edition2026']
            variants = ['default', 'purego'] if legacy else ['default', 'purego', 'race']
            checks = ['go-version', 'compiler-version', 'download-root', 'install-generators']
            if not legacy:
                checks += ['download-reflect', 'download-runtime-pin', 'stable-override', 'stable-metadata-control']
            report = {'status': 'passed', 'revision': self.revision, 'public_commit': self.revision,
                      'repository': self.manifest['repository'], 'manifest_sha256': self.digest,
                      'cold_module_cache': True, 'published_version_mode': False,
                      'go': 'go version go1.' + str(minor) + '.0 linux/amd64',
                      'profile': 'legacy' if legacy else 'legacy-and-native',
                      'consumer_runs': [{'backend': b, 'profile': f, 'variant': v}
                                        for b in plan.BACKENDS for f in flavors for v in variants],
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
            self.reports.append(report)

    def test_rejects_unqualified_publication_evidence(self):
        plan.qualify(self.reports, self.manifest, self.digest, self.revision)
        for kind in ['revision', 'failed', 'manifest', 'matrix', 'duplicate', 'checksum', 'missing-control', 'missing-command', 'mode', 'control-crash', 'missing-go115']:
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
            else:
                bad.pop(0)
            with self.subTest(kind=kind), self.assertRaises(ValueError):
                plan.qualify(bad, self.manifest, self.digest, self.revision)


if __name__ == '__main__':
    unittest.main()
