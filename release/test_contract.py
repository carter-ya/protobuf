import copy
import importlib.util
import json
import pathlib
import unittest

HERE = pathlib.Path(__file__).resolve().parent
spec = importlib.util.spec_from_file_location('contract', HERE / 'check.py')
contract = importlib.util.module_from_spec(spec)
spec.loader.exec_module(contract)


class ReleaseContract(unittest.TestCase):
    def setUp(self):
        self.manifest = json.loads((HERE / 'manifest.json').read_text())
        self.root_mod = (HERE.parent / 'go.mod').read_text()
        self.reflect_mod = (HERE.parent / 'reflect/go.mod').read_text()
        self.sums = (HERE.parent / 'reflect/go.sum').read_text()

    def test_rejects_version_and_runtime_drift(self):
        contract.validate(self.manifest, self.root_mod, self.reflect_mod, self.sums)
        cases = [('root', 'go', '1.23'), ('reflect', 'tag', 'v0.1.0-rc.1'),
                 ('reflect', 'minimum_root', 'v1.3.2'),
                 ('official_runtime', 'version', 'v1.36.12'),
                 ('official_runtime', 'sum', 'h1:wrong'),
                 ('official_runtime', 'commit_prefix', '0123456789ab'),
                 ('official_runtime', 'stable_reference_gap', {'kind': 'any-failure', 'diagnostic': 'FAIL'})]
        for section, field, value in cases:
            bad = copy.deepcopy(self.manifest)
            bad[section][field] = value
            with self.subTest(section=section, field=field), self.assertRaises(ValueError):
                contract.validate(bad, self.root_mod, self.reflect_mod, self.sums)

    def test_rejects_unpaired_minimum_and_root_dependency(self):
        bad = self.reflect_mod.replace('github.com/gogo/protobuf ' + self.manifest['root']['version'],
                                      'github.com/gogo/protobuf v1.3.2')
        with self.assertRaises(ValueError):
            contract.validate(self.manifest, self.root_mod, bad, self.sums)
        with self.assertRaises(ValueError):
            contract.validate(self.manifest, self.root_mod + '\nrequire google.golang.org/protobuf v1.36.12\n', self.reflect_mod, self.sums)

    def test_rejects_incomplete_published_provenance(self):
        for kind in ['revision', 'tag', 'missing-module', 'path', 'version', 'sum', 'go-mod-sum']:
            bad = copy.deepcopy(self.manifest)
            pair = bad['published_pair']
            if kind == 'revision':
                pair['revision'] = 'main'
            elif kind == 'tag':
                pair['reflect'] = pair['reflect'].removeprefix('reflect/')
            elif kind == 'missing-module':
                pair['modules'].pop('reflect')
            elif kind == 'path':
                pair['modules']['root']['Path'] = 'github.com/gogo/protobuf'
            elif kind == 'version':
                pair['modules']['reflect']['Version'] = 'v0.0.0'
            else:
                pair['modules']['root']['Sum' if kind == 'sum' else 'GoModSum'] = 'h1:wrong'
            with self.subTest(kind=kind), self.assertRaises(ValueError):
                contract.validate(bad, self.root_mod, self.reflect_mod, self.sums)


if __name__ == '__main__':
    unittest.main()
