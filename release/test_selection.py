import copy
import json
import pathlib
import unittest

import check
import selection


class ComponentSelection(unittest.TestCase):
    def setUp(self):
        here = pathlib.Path(__file__).resolve().parent
        self.here = here
        self.manifest = json.loads((here / 'manifest.json').read_text())
        self.revision = '0123456789abcdef0123456789abcdef01234567'

    def test_root_only_promotion_keeps_frozen_modern_origin_and_checksum(self):
        origins = selection.component_revisions(self.manifest, self.revision)
        self.assertEqual(origins['root'], self.revision)
        self.assertEqual(origins['reflect'], selection.published_revisions(self.manifest['published_pair'])['reflect'])
        self.assertEqual(selection.publish_components(self.manifest), ['root'])
        data = copy.deepcopy(self.manifest['published_pair']['modules']['reflect'])
        selection.check_frozen_identity(self.manifest, 'reflect', data)
        for field in ['Version', 'Sum', 'GoModSum']:
            bad = copy.deepcopy(data)
            bad[field] = 'changed'
            with self.subTest(field=field), self.assertRaises(ValueError):
                selection.check_frozen_identity(self.manifest, 'reflect', bad)

    def test_paired_rc_mode_still_selects_both_candidate_components(self):
        paired = copy.deepcopy(self.manifest)
        paired['state'] = 'release-candidate'
        self.assertEqual(selection.component_revisions(paired, self.revision), {'root': self.revision, 'reflect': self.revision})
        self.assertEqual(selection.publish_components(paired), ['root', 'reflect'])

    def test_component_records_can_keep_two_distinct_revisions(self):
        pair = copy.deepcopy(self.manifest['published_pair'])
        pair['revision'] = self.revision
        pair['revisions']['root'] = self.revision
        self.assertNotEqual(selection.published_revisions(pair)['root'], selection.published_revisions(pair)['reflect'])
        pair['revisions']['root'] = 'main'
        with self.assertRaises(ValueError):
            selection.published_revisions(pair)

    def test_frozen_go_mod_and_minimum_root_are_enforced(self):
        root_mod = (self.here.parent / 'go.mod').read_text()
        modern_mod = (self.here.parent / 'reflect/go.mod').read_text()
        sums = (self.here.parent / 'reflect/go.sum').read_text()
        self.assertEqual(selection.go_mod_sum(modern_mod.encode()), self.manifest['published_pair']['modules']['reflect']['GoModSum'])
        with self.assertRaises(ValueError):
            check.validate(self.manifest, root_mod, modern_mod + '\n// changed\n', sums)
        bad = copy.deepcopy(self.manifest)
        bad['root']['version'] = bad['root']['tag'] = 'v1.3.0'
        with self.assertRaises(ValueError):
            check.validate(bad, root_mod, modern_mod, sums)


if __name__ == '__main__':
    unittest.main()
