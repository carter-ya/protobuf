"""Remote provenance for the published-pair real-project matrix."""
import json
import os
import pathlib
import re
import sys
import urllib.parse
import urllib.request

ROOT = pathlib.Path(__file__).resolve().parents[3]
sys.path.insert(0, str(ROOT / 'release'))
from remote import objects, validate_graph as validate_release_graph
from selection import component_revisions, published_revisions, root_stable


def environment(destination):
    env = dict(os.environ, GOENV='off', GOWORK='off', GOTOOLCHAIN='local', GOFLAGS='',
               GOMODCACHE=str(destination / 'modules'), GOCACHE=str(destination / 'build-cache'),
               GOPATH=str(destination / 'gopath'), GOBIN=str(destination / 'bin'),
               GOPROXY='https://proxy.golang.org,direct', GOSUMDB='sum.golang.org',
               GOPRIVATE='', GONOPROXY='', GONOSUMDB='', GOINSECURE='', GOAUTH='off', GIT_TERMINAL_PROMPT='0',
               GIT_CONFIG_GLOBAL='/dev/null', GIT_CONFIG_SYSTEM='/dev/null')
    for key in list(env):
        if key in ['GH_TOKEN', 'GITHUB_TOKEN', 'SSH_AUTH_SOCK', 'GIT_SSH', 'GIT_SSH_COMMAND',
                   'GOCACHEPROG', 'GIT_ASKPASS', 'SSH_ASKPASS', 'GIT_CONFIG', 'GIT_CONFIG_PARAMETERS'] or key.startswith(('GIT_CONFIG_KEY_', 'GIT_CONFIG_VALUE_')) or key == 'GIT_CONFIG_COUNT':
            env.pop(key, None)
    return env


def check_download(data, identity, cache, revision=None):
    if any(data.get(k) != identity[k] for k in ['Path', 'Version', 'Sum', 'GoModSum']):
        raise ValueError('published module version or checksum differs: ' + identity['Path'])
    if not pathlib.Path(data['Dir']).resolve().is_relative_to(cache.resolve()):
        raise ValueError('downloaded source is outside the fresh cache')
    if revision is not None and data.get('Origin', {}).get('Hash') != revision:
        raise ValueError('published module origin differs')


def check_graph(graph, manifest, versions, cache, original=None, local_project=None, repository=None):
    """Allow only the named original control and regenerated project copies locally."""
    allowed = dict(local_project or {})
    if original is not None:
        allowed['github.com/gogo/protobuf'] = original
    filtered, seen = [], set()
    for module in graph:
        name = module['Path']
        if name in allowed:
            replacement = module.get('Replace', {})
            if replacement.get('Version') or replacement.get('Path') != str(allowed[name]) or pathlib.Path(replacement.get('Dir', '')).resolve() != allowed[name].resolve():
                raise ValueError('unexpected local control/project replacement: ' + name)
            if name == 'github.com/gogo/protobuf' and module.get('Version') != 'v1.3.2':
                raise ValueError('original control selected a different runtime version')
            seen.add(name)
        else:
            filtered.append(module)
    if seen != set(allowed):
        raise ValueError('declared local control/project missing from module graph')
    if original is None:
        validate_release_graph(filtered, manifest, versions, cache, repository or manifest['repository'], True)
    else:
        if any(m['Path'] == 'github.com/gogo/protobuf/reflect' for m in filtered):
            raise ValueError('original control acquired the maintained reflection module')
        for module in filtered:
            replacement = module.get('Replace')
            selected = replacement or module
            if replacement and not replacement.get('Version'):
                raise ValueError('unqualified local dependency in original control')
            if not module.get('Main') and selected.get('Dir') and not pathlib.Path(selected['Dir']).resolve().is_relative_to(cache.resolve()):
                raise ValueError('dependency source outside fresh cache')


class PublishedPair:
    def __init__(self, destination, run, report, candidate_root=None, candidate_repository=None):
        self.destination, self.run = destination, run
        self.cache = destination / 'modules'
        self.cache.mkdir()
        record = json.loads((ROOT / 'release/manifest.json').read_text())
        self.pair = record['published_pair']
        self.repository = record['repository']
        self.root_repository = candidate_repository or self.repository
        if not re.fullmatch(r'github\.com/[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+', self.root_repository):
            raise ValueError('candidate repository must be a public GitHub owner/repo')
        if candidate_root and not root_stable(record):
            raise ValueError('candidate-root mode requires the independent stable-root contract')
        self.versions = {name: data['Version'] for name, data in self.pair['modules'].items()}
        origins = published_revisions(self.pair)
        if candidate_root:
            origins = component_revisions(record, candidate_root)
            self.versions['root'] = record['root']['version']
        modules, commits = {}, {}
        for name in ['root', 'reflect']:
            candidate = bool(candidate_root and name == 'root')
            ref = candidate_root if candidate else self.pair[name]
            repository = self.root_repository if candidate else self.repository
            url = 'https://api.github.com/repos/' + repository.removeprefix('github.com/') + '/commits/' + urllib.parse.quote(ref, safe='')
            request = urllib.request.Request(url, headers={'User-Agent': 'protobuf-published-downstream'})
            if os.environ.get('GITHUB_TOKEN'):
                request.add_header('Authorization', 'Bearer ' + os.environ['GITHUB_TOKEN'])
            with urllib.request.urlopen(request, timeout=60) as response:
                commits[name] = json.load(response)['sha']
            if commits[name] != origins[name]:
                raise ValueError('component public origin differs from the pinned revision')
            path = self.root_repository if candidate else self.pair['modules'][name]['Path']
            query = candidate_root if candidate else self.pair['modules'][name]['Version']
            result = run('published-download-' + name, ['go', 'mod', 'download', '-json', path + '@' + query], capture=True)
            data = json.loads(result.stdout)
            identity = self.pair['modules'][name]
            if candidate:
                if data['Path'] != path or not data.get('Sum') or not data.get('GoModSum'):
                    raise ValueError('candidate module identity or checksums missing')
                identity = {k: data[k] for k in ['Path', 'Version', 'Sum', 'GoModSum']}
            check_download(data, identity, self.cache, origins[name])
            canonical = 'github.com/gogo/protobuf' + ('/reflect' if name == 'reflect' else '')
            declaration = re.search(r'^module\s+(\S+)', pathlib.Path(data['GoMod']).read_text(), re.M)
            if not declaration or declaration.group(1) != canonical:
                raise ValueError('published canonical module declaration differs')
            modules[name] = data
        self.root = pathlib.Path(modules['root']['Dir'])
        self.manifest = json.loads((self.root / 'release/manifest.json').read_text())
        if candidate_root and self.manifest != record:
            raise ValueError('candidate source contract differs from reviewed harness contract')
        if self.manifest['repository'] != self.repository:
            raise ValueError('published source repository differs')
        for name in ['root', 'reflect']:
            tag = record['root']['tag'] if candidate_root and name == 'root' else self.pair[name]
            if self.manifest[name]['tag'] != tag or self.manifest[name]['version'] != self.versions[name]:
                raise ValueError('published source release contract differs from its tag')
        runtime = self.manifest['official_runtime']
        self.runtime = runtime['version']
        result = run('published-download-runtime', ['go', 'mod', 'download', '-json', runtime['module'] + '@' + runtime['version']], capture=True)
        data = json.loads(result.stdout)
        check_download(data, {'Path': runtime['module'], 'Version': self.runtime, 'Sum': runtime['sum'], 'GoModSum': runtime['go_mod_sum']}, self.cache)
        report['published_pair'] = self.pair
        report['tested_revision'] = origins['root']
        report['tested_revisions'] = origins
        report['public_component_commits'] = commits
        report['published_tag_commits'] = {name: value for name, value in commits.items() if not candidate_root or name != 'root'}
        report['selected_modules'] = {k: {f: v.get(f) for f in ['Path', 'Version', 'Sum', 'GoModSum', 'Dir', 'Origin']} for k, v in modules.items()}
        if not candidate_root:
            report['published_modules'] = report['selected_modules']
        else:
            report['candidate_root'] = candidate_root
            report['candidate_repository'] = self.root_repository
            report['canonical_versions'] = self.versions
        report['published_runtime'] = {f: data.get(f) for f in ['Path', 'Version', 'Sum', 'GoModSum', 'Dir']}
        self.tools = destination / 'tools'
        self.modules = modules
        self.downloaded_versions = {name: data['Version'] for name, data in modules.items()}
        self.tools.mkdir()
        (self.tools / 'go.mod').write_text('module example.com/published-gogo-tools\n\ngo 1.26.0\n\nrequire github.com/gogo/protobuf ' + self.versions['root'] + '\nreplace github.com/gogo/protobuf => ' + self.replacement('root') + '\n')

    def replacement(self, component):
        return self.modules[component]['Path'] + ' ' + self.downloaded_versions[component]

    def graph(self, name, consumer, report, original=None, local_project=None, modern=True):
        result = self.run(name, ['go', 'list', '-mod=mod', '-m', '-json', 'all'], consumer, capture=True)
        graph = objects(result.stdout)
        if not modern:
            validate_release_graph(graph, self.manifest, self.downloaded_versions, self.cache, self.root_repository, False)
        else:
            check_graph(graph, self.manifest, self.downloaded_versions, self.cache, original, local_project, self.root_repository)
        report.setdefault('module_graphs', []).append({'name': name, 'graph': graph})

    def generator(self, backend, report):
        self.run('published-install-' + backend, ['go', 'install', '-mod=mod', 'github.com/gogo/protobuf/protoc-gen-' + backend], self.tools)
        driver = self.destination / 'bin' / ('protoc-gen-' + backend)
        result = self.run('published-build-info-' + backend, ['go', 'version', '-m', driver], capture=True)
        data = self.modules['root']
        expected = r'\t(?:mod|dep)\tgithub\.com/gogo/protobuf\t' + re.escape(self.versions['root']) + r'\s*\n\t=>\t' + re.escape(data['Path']) + r'\t' + re.escape(data['Version']) + r'\t' + re.escape(data['Sum'])
        if not re.search(expected, result.stdout):
            raise ValueError('generator binary does not identify the qualified published root')
        self.graph('published-tools-graph-' + backend, self.tools, report, modern=False)
        return driver
