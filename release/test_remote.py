import copy
import importlib.util
import json
import pathlib
import tempfile
import unittest

HERE = pathlib.Path(__file__).resolve().parent
spec = importlib.util.spec_from_file_location('remote', HERE / 'remote.py')
remote = importlib.util.module_from_spec(spec)
spec.loader.exec_module(remote)


class RemoteSourceGuards(unittest.TestCase):
    def test_rejects_local_sources_mixed_versions_and_runtime_override(self):
        manifest = json.loads((HERE / 'manifest.json').read_text())
        with tempfile.TemporaryDirectory() as temporary:
            cache = pathlib.Path(temporary) / 'modules'
            cache.mkdir()
            versions = {'root': 'v1.3.3-0.20261101144239-0123456789ab',
                        'reflect': 'v0.0.0-20261101144239-0123456789ab'}
            repository = manifest['repository']
            graph = [{'Path': 'example.com/release-consumer', 'Main': True, 'Dir': str(HERE)},
                     {'Path': manifest['root']['module'], 'Version': manifest['root']['version'],
                      'Replace': {'Path': repository, 'Version': versions['root'], 'Dir': str(cache / 'root')}},
                     {'Path': manifest['reflect']['module'], 'Version': manifest['reflect']['version'],
                      'Replace': {'Path': repository + '/reflect', 'Version': versions['reflect'], 'Dir': str(cache / 'reflect')}},
                     {'Path': manifest['official_runtime']['module'],
                      'Version': manifest['official_runtime']['version'], 'Dir': str(cache / 'runtime')}]
            remote.validate_graph(graph, manifest, versions, cache, repository, True)
            for kind in ['root-path', 'reflect-local', 'root-version', 'runtime-version', 'runtime-replace', 'other-local']:
                bad = copy.deepcopy(graph)
                if kind == 'root-path':
                    bad[1]['Replace']['Dir'] = str(HERE)
                elif kind == 'reflect-local':
                    bad[2]['Replace'].pop('Version')
                elif kind == 'root-version':
                    bad[1]['Replace']['Version'] = 'v1.3.2'
                elif kind == 'runtime-version':
                    bad[3]['Version'] = manifest['official_runtime']['stable_reference']
                elif kind == 'runtime-replace':
                    bad[3]['Replace'] = {'Path': bad[3]['Path'], 'Version': 'v1.36.12', 'Dir': str(cache / 'stable')}
                else:
                    bad.append({'Path': 'example.com/hidden', 'Version': 'v1.0.0', 'Dir': str(HERE)})
                with self.subTest(kind=kind), self.assertRaises(ValueError):
                    remote.validate_graph(bad, manifest, versions, cache, repository, True)

    def test_module_json_stream_keeps_every_object(self):
        self.assertEqual(remote.objects(' {"Path":"one"}\n {"Path":"two"}\n'),
                         [{'Path': 'one'}, {'Path': 'two'}])
        with self.assertRaises(ValueError):
            remote.objects('{"Path":"one"}\n malformed')


if __name__ == '__main__':
    unittest.main()
