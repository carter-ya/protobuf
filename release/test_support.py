import copy
import json
import pathlib
import unittest

import support


class SupportLine(unittest.TestCase):
    def setUp(self):
        here = pathlib.Path(__file__).resolve().parent
        self.policy = json.loads((here / 'support-policy.json').read_text())
        self.manifest = json.loads((here / 'manifest.json').read_text())

    def test_current_line_and_independent_root_eligibility(self):
        support.validate(self.policy, self.manifest)
        candidate = copy.deepcopy(self.manifest)
        candidate['root']['version'] = 'v1.4.0'
        support.validate(self.policy, candidate)

    def test_rejects_silent_toolchain_or_channel_promotion(self):
        for kind in ['root-go', 'modern-go', 'modern-stable', 'runtime-policy']:
            bad = copy.deepcopy(self.manifest)
            if kind == 'root-go':
                bad['root']['go'] = '1.23'
            elif kind == 'modern-go':
                bad['reflect']['go'] = '1.26'
            elif kind == 'modern-stable':
                bad['reflect']['version'] = 'v0.1.0'
            else:
                bad['official_runtime']['policy'] = 'automatic-latest'
            with self.subTest(kind=kind), self.assertRaises(ValueError):
                support.validate(self.policy, bad)

    def test_rejects_dropping_snapshot_stability_or_provenance_gates(self):
        for section, key in [('root', 'official_v2_dependency'), ('reflect', 'stable_requires_qualified_official_stable_runtime'),
                             ('pairing', 'immutable_published_components'), ('pairing', 'per_component_revision_and_checksum_evidence')]:
            bad = copy.deepcopy(self.policy)
            bad[section][key] = not bad[section][key]
            with self.subTest(key=key), self.assertRaises(ValueError):
                support.validate(bad, self.manifest)


if __name__ == '__main__':
    unittest.main()
