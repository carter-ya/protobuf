"""Unit fixtures for provenance rejection; these are not downstream evidence."""
import copy
import json
import pathlib
import tempfile
import unittest

import published


class PublishedProvenance(unittest.TestCase):
    def setUp(self):
        self.temporary = tempfile.TemporaryDirectory()
        self.addCleanup(self.temporary.cleanup)
        self.out = pathlib.Path(self.temporary.name)
        self.cache = self.out / 'modules'
        self.cache.mkdir()
        self.record = json.loads((published.ROOT / 'release/manifest.json').read_text())
        self.pair = self.record['published_pair']
        self.versions = {n: m['Version'] for n, m in self.pair['modules'].items()}
        self.manifest = copy.deepcopy(self.record)
        self.manifest['root']['version'] = self.versions['root']
        self.manifest['reflect']['version'] = self.versions['reflect']
        self.graph = [{'Path': 'example.com/consumer', 'Main': True, 'Dir': str(self.out / 'consumer')}]
        for name in ['root', 'reflect']:
            self.graph.append({'Path': self.manifest[name]['module'], 'Version': self.versions[name],
                'Replace': {'Path': self.pair['modules'][name]['Path'], 'Version': self.versions[name], 'Dir': str(self.cache / name)}})
        self.graph.append({'Path': self.manifest['official_runtime']['module'],
            'Version': self.manifest['official_runtime']['version'], 'Dir': str(self.cache / 'runtime')})

    def test_rejects_wrong_artifact_checksum_revision_and_local_source(self):
        identity = self.pair['modules']['root']
        data = dict(identity, Dir=str(self.cache / 'root'), Origin={'Hash': self.pair['revision']})
        published.check_download(data, identity, self.cache, self.pair['revision'])
        for key, value in [('Path', 'github.com/gogo/protobuf'), ('Version', 'v1.3.2'), ('Sum', 'h1:wrong'),
                           ('GoModSum', 'h1:wrong'), ('Dir', str(published.ROOT)), ('Origin', {'Hash': '0' * 40})]:
            bad = copy.deepcopy(data)
            bad[key] = value
            with self.subTest(key=key), self.assertRaises(ValueError):
                published.check_download(bad, identity, self.cache, self.pair['revision'])

    def test_only_the_named_regenerated_project_may_be_local(self):
        source = self.out / 'upstream-regenerated'
        graph = self.graph + [{'Path': 'example.com/upstream', 'Version': 'v1.2.3',
            'Replace': {'Path': str(source), 'Dir': str(source)}}]
        allowed = {'example.com/upstream': source}
        published.check_graph(graph, self.manifest, self.versions, self.cache, local_project=allowed)
        for kind in ['other-local', 'fork-local', 'wrong-project', 'root-version', 'runtime-version', 'runtime-override']:
            bad = copy.deepcopy(graph)
            if kind == 'other-local':
                bad.append({'Path': 'example.com/hidden', 'Replace': {'Path': str(source), 'Dir': str(source)}})
            elif kind == 'fork-local':
                bad[1]['Replace'] = {'Path': str(published.ROOT), 'Dir': str(published.ROOT)}
            elif kind == 'wrong-project':
                bad[-1]['Replace']['Dir'] = str(published.ROOT)
            elif kind == 'root-version':
                bad[1]['Replace']['Version'] = 'v1.3.2'
            elif kind == 'runtime-version':
                bad[3]['Version'] = 'v1.36.12'
            else:
                bad[3]['Replace'] = {'Path': bad[3]['Path'], 'Version': bad[3]['Version'], 'Dir': str(self.cache / 'other-runtime')}
            with self.subTest(kind=kind), self.assertRaises(ValueError):
                published.check_graph(bad, self.manifest, self.versions, self.cache, local_project=allowed)

    def test_original_control_is_explicit_and_cannot_contaminate_the_fork(self):
        original = self.out / 'original'
        graph = [self.graph[0], {'Path': 'github.com/gogo/protobuf', 'Version': 'v1.3.2',
                 'Replace': {'Path': str(original), 'Dir': str(original)}}]
        published.check_graph(graph, self.manifest, self.versions, self.cache, original=original)
        bad = copy.deepcopy(graph)
        bad[1]['Version'] = self.versions['root']
        with self.assertRaises(ValueError):
            published.check_graph(bad, self.manifest, self.versions, self.cache, original=original)
        with self.assertRaises(ValueError):
            published.check_graph(graph + [self.graph[2]], self.manifest, self.versions, self.cache, original=original)
        with self.assertRaises(ValueError):
            published.check_graph(graph + [self.graph[2], self.graph[3]], self.manifest, self.versions, self.cache)

    def test_fresh_environment_ignores_ambient_replacements_and_credentials(self):
        from unittest.mock import patch
        with patch.dict('os.environ', {'GOFLAGS': '-modfile=/tmp/injected.mod', 'GOWORK': '/tmp/work',
                'GH_TOKEN': 'unit-only', 'GITHUB_TOKEN': 'unit-only', 'SSH_AUTH_SOCK': '/tmp/socket',
                'GIT_CONFIG_COUNT': '1', 'GIT_CONFIG_KEY_0': 'url.fake.insteadOf', 'GIT_CONFIG_VALUE_0': 'fake',
                'GIT_CONFIG_PARAMETERS': 'unit-only', 'GIT_ASKPASS': '/tmp/askpass', 'GOCACHEPROG': '/tmp/cache', 'GOAUTH': 'netrc'}):
            env = published.environment(self.out)
        self.assertEqual(env['GOFLAGS'], '')
        self.assertEqual(env['GOWORK'], 'off')
        self.assertEqual(env['GOMODCACHE'], str(self.cache))
        self.assertEqual(env['GOAUTH'], 'off')
        for name in ['GH_TOKEN', 'GITHUB_TOKEN', 'SSH_AUTH_SOCK', 'GIT_CONFIG_COUNT', 'GIT_CONFIG_KEY_0', 'GIT_CONFIG_VALUE_0',
                     'GIT_CONFIG_PARAMETERS', 'GIT_ASKPASS', 'GOCACHEPROG']:
            self.assertNotIn(name, env)


if __name__ == '__main__':
    unittest.main()
