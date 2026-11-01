#!/usr/bin/env python3
"""Qualify public remote modules in an external consumer with an empty cache."""
import argparse
import datetime
import hashlib
import json
import os
import pathlib
import re
import shutil
import subprocess
import urllib.request
import urllib.parse

ROOT = pathlib.Path(__file__).resolve().parents[1]
BACKENDS = ['gogo', 'gogofast', 'gogofaster', 'gogoslick', 'gogotypes']


def objects(data):
    decoder, result = json.JSONDecoder(), []
    while data.strip():
        item, end = decoder.raw_decode(data.lstrip())
        result.append(item)
        data = data.lstrip()[end:]
    return result


def validate_graph(graph, manifest, versions, cache, repository, modern):
    cache = cache.resolve()
    by_name = {m['Path']: m for m in graph}
    for component in ['root', 'reflect'] if modern else ['root']:
        name = manifest[component]['module']
        if by_name[name].get('Version') != manifest[component]['version']:
            raise ValueError('canonical module version differs from candidate pair')
        replacement = by_name[name].get('Replace', {})
        target = repository + ('/reflect' if component == 'reflect' else '')
        if replacement.get('Path') != target or replacement.get('Version') != versions[component]:
            raise ValueError('remote module replacement drift: ' + name)
        if not pathlib.Path(replacement['Dir']).resolve().is_relative_to(cache):
            raise ValueError('source is outside the fresh module cache')
    for module in graph:
        replacement = module.get('Replace')
        if replacement and not replacement.get('Version'):
            raise ValueError('local-path replacement in consumer graph: ' + module['Path'])
        selected = replacement or module
        if not module.get('Main') and selected.get('Dir') and not pathlib.Path(selected['Dir']).resolve().is_relative_to(cache):
            raise ValueError('dependency source outside fresh cache: ' + module['Path'])
    if modern:
        selected = by_name[manifest['official_runtime']['module']]
        if selected.get('Replace') or selected['Version'] != manifest['official_runtime']['version']:
            raise ValueError('unqualified official runtime selected')


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('output', type=pathlib.Path)
    parser.add_argument('--revision', required=True, help='exact public 40-character commit SHA')
    parser.add_argument('--repository', default='github.com/carter-ya/protobuf')
    parser.add_argument('--legacy-only', action='store_true')
    parser.add_argument('--backend', action='append', choices=BACKENDS)
    parser.add_argument('--published', action='store_true', help='resolve the manifest version pair instead of commit queries')
    parser.add_argument('--probe-stable', action='store_true', help='record the pinned stable-reference feature gap')
    args = parser.parse_args()
    if args.probe_stable and args.legacy_only:
        parser.error('stable probe requires the modern profile')
    if not re.fullmatch(r'[0-9a-f]{40}', args.revision):
        parser.error('revision must be an immutable full commit SHA')
    if not re.fullmatch(r'github\.com/[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+', args.repository):
        parser.error('repository must be a public GitHub owner/repo module path')
    out = args.output.resolve()
    if out.exists() or out.is_relative_to(ROOT):
        parser.error('use a new output directory outside the source checkout')
    out.mkdir(parents=True)
    cache = out / 'modules'
    cache.mkdir()
    manifest = json.loads((ROOT / 'release/manifest.json').read_text())
    report = {'status': 'running', 'time': datetime.datetime.now(datetime.timezone.utc).isoformat(),
              'revision': args.revision, 'repository': args.repository,
              'published_version_mode': args.published, 'cold_module_cache': True,
              'profile': 'legacy' if args.legacy_only else 'legacy-and-native',
              'manifest_sha256': hashlib.sha256((ROOT / 'release/manifest.json').read_bytes()).hexdigest(),
              'checks': [], 'modules': {}, 'consumer_runs': []}
    env = dict(os.environ, GOENV='off', GOWORK='off', GOTOOLCHAIN='local', GOFLAGS='',
               GOMODCACHE=str(cache), GOCACHE=str(out / 'build-cache'), GOPATH=str(out / 'gopath'),
               GOBIN=str(out / 'bin'), GOPROXY='https://proxy.golang.org,direct', GOSUMDB='sum.golang.org',
               GOPRIVATE='', GONOPROXY='', GONOSUMDB='', GIT_TERMINAL_PROMPT='0',
               GIT_CONFIG_GLOBAL='/dev/null', GIT_CONFIG_SYSTEM='/dev/null')
    for key in list(env):
        if key in ['GH_TOKEN', 'GITHUB_TOKEN', 'SSH_AUTH_SOCK', 'GIT_SSH', 'GIT_SSH_COMMAND'] or key.startswith('GIT_CONFIG_KEY_') or key.startswith('GIT_CONFIG_VALUE_') or key == 'GIT_CONFIG_COUNT':
            env.pop(key, None)

    def save():
        (out / 'qualification.json').write_text(json.dumps(report, indent=2) + '\n')

    def run(name, command, cwd=out, allow_failure=False):
        result = subprocess.run([str(x) for x in command], cwd=cwd, env=env, text=True,
                                stdout=subprocess.PIPE, stderr=subprocess.PIPE)
        (out / (name + '.log')).write_text(result.stdout + result.stderr)
        report['checks'].append({'name': name, 'command': [str(x) for x in command],
                                 'returncode': result.returncode, 'expected_failure': allow_failure})
        save()
        if result.returncode and not allow_failure:
            raise RuntimeError(name + ' failed; inspect its log')
        print(name, 'finished:', result.returncode, flush=True)
        return result

    versions = {}

    def modfile(dest, modern):
        dest.mkdir(parents=True)
        components = ['root', 'reflect'] if modern else ['root']
        text = 'module example.com/release-consumer\n\ngo ' + ('1.23' if modern else '1.15') + '\nrequire (\n'
        for component in components:
            text += ' ' + manifest[component]['module'] + ' ' + manifest[component]['version'] + '\n'
        text += ')\n'
        for component in components:
            text += 'replace ' + manifest[component]['module'] + ' => ' + args.repository
            text += ('/reflect' if component == 'reflect' else '') + ' ' + versions[component] + '\n'
        (dest / 'go.mod').write_text(text)

    save()
    try:
        report['go'] = run('go-version', ['go', 'version']).stdout.strip()
        report['compiler'] = run('compiler-version', ['protoc', '--version']).stdout.strip()
        if report['compiler'] != 'libprotoc ' + manifest['compiler']:
            raise ValueError('remote consumer gate requires the manifest compiler')
        go_minor = int(re.search(r'go1\.(\d+)', report['go']).group(1))
        if go_minor < (15 if args.legacy_only else 23):
            raise ValueError('toolchain below profile minimum')
        def public_commit(ref):
            url = ('https://api.github.com/repos/' + args.repository.removeprefix('github.com/')
                   + '/commits/' + urllib.parse.quote(ref, safe=''))
            request = urllib.request.Request(url, headers={'User-Agent': 'protobuf-release-preflight'})
            # Optional CI authentication is used only for metadata lookup. All
            # Go/git consumer subprocesses have tokens and SSH access removed.
            metadata_token = os.environ.get('GITHUB_TOKEN')
            if metadata_token:
                request.add_header('Authorization', 'Bearer ' + metadata_token)
            with urllib.request.urlopen(request, timeout=60) as response:
                return json.load(response)['sha']

        report['public_commit'] = public_commit(args.revision)
        if report['public_commit'] != args.revision:
            raise ValueError('public revision lookup mismatch')
        components = ['root'] if args.legacy_only else ['root', 'reflect']
        for component in components:
            path = args.repository + ('/reflect' if component == 'reflect' else '')
            query = manifest[component]['version'] if args.published else args.revision
            if args.published:
                tag_commit = public_commit(manifest[component]['tag'])
                if tag_commit != args.revision:
                    raise ValueError('published module tags do not share the selected commit')
                report.setdefault('published_tag_commits', {})[component] = tag_commit
            data = json.loads(run('download-' + component, ['go', 'mod', 'download', '-json', path + '@' + query]).stdout)
            if not pathlib.Path(data['Dir']).resolve().is_relative_to(cache):
                raise ValueError('downloaded module is outside the fresh cache')
            if not data.get('Sum') or not data.get('GoModSum'):
                raise ValueError('remote module checksum unavailable')
            origin = data.get('Origin', {})
            if origin.get('Hash'):
                if origin['Hash'] != args.revision:
                    raise ValueError('module resolved to a different revision')
            elif not args.published and not data['Version'].endswith('-' + args.revision[:12]):
                raise ValueError('old-toolchain pseudo-version does not identify revision')
            if data['Path'] != path or args.published and data['Version'] != query:
                raise ValueError('downloaded module identity mismatch')
            declaration = re.search(r'^module\s+(\S+)', pathlib.Path(data['GoMod']).read_text(), re.M)
            if not declaration or declaration.group(1) != manifest[component]['module']:
                raise ValueError('downloaded canonical module declaration differs')
            versions[component] = data['Version']
            report['modules'][component] = {k: data.get(k) for k in ['Path', 'Version', 'Sum', 'GoModSum', 'Dir', 'Origin']}
        root_dir = pathlib.Path(report['modules']['root']['Dir'])
        if json.loads((root_dir / 'release/manifest.json').read_text()) != manifest:
            raise ValueError('downloaded candidate contract differs from checked contract')
        tools = out / 'tools'
        modfile(tools, False)
        backends = args.backend or BACKENDS
        run('install-generators', ['go', 'install', '-mod=mod'] + [manifest['root']['module'] + '/protoc-gen-' + name for name in backends], tools)
        for backend in backends:
            info = run('build-info-' + backend, ['go', 'version', '-m', out / 'bin' / ('protoc-gen-' + backend)]).stdout
            if args.repository not in info or versions['root'] not in info:
                raise ValueError('generator identity is not the downloaded fork')
        graph = objects(run('tools-modules', ['go', 'list', '-mod=mod', '-m', '-json', 'all'], tools).stdout)
        validate_graph(graph, manifest, versions, cache, args.repository, False)
        flavors = ['legacy'] if args.legacy_only else ['legacy', 'open', 'hybrid', 'edition2026']
        stable_source = None
        for backend in backends:
            for flavor in flavors:
                modern = flavor != 'legacy'
                dest = out / 'consumers' / (backend + '-' + flavor)
                modfile(dest, modern)
                (dest / 'pb').mkdir()
                schema = 'hybrid' if flavor == 'open' else flavor
                shutil.copyfile(ROOT / 'release/consumer' / (schema + '.proto'), dest / 'sample.proto')
                params = 'paths=source_relative' + (',default_api_level=API_HYBRID' if flavor == 'hybrid' else ',proto_reflect=true' if flavor == 'open' else '')
                run(backend + '-' + flavor + '-generate', ['protoc', '-I' + str(dest), '-I' + str(root_dir),
                    '-I' + str(root_dir / 'protobuf'), '--plugin=protoc-gen-test=' + str(out / 'bin' / ('protoc-gen-' + backend)),
                    '--test_out=' + params + ':' + str(dest / 'pb'), 'sample.proto'])
                shutil.copyfile(ROOT / 'release/consumer' / (flavor + '_test.go.txt'), dest / 'consumer_test.go')
                for variant, flags in [('default', []), ('purego', ['-tags=purego'])] + ([] if args.legacy_only else [('race', ['-race'])]):
                    run(backend + '-' + flavor + '-' + variant, ['go', 'test', '-mod=mod', '-count=1'] + flags + ['./...'], dest)
                    report['consumer_runs'].append({'backend': backend, 'profile': flavor, 'variant': variant})
                graph = objects(run(backend + '-' + flavor + '-modules', ['go', 'list', '-mod=mod', '-m', '-json', 'all'], dest).stdout)
                validate_graph(graph, manifest, versions, cache, args.repository, modern)
                if not modern:
                    deps = run(backend + '-legacy-imports', ['go', 'list', '-mod=mod', '-deps', './...'], dest).stdout.splitlines()
                    if any(p.startswith('google.golang.org/protobuf/') or p.startswith('github.com/gogo/protobuf/reflect') for p in deps):
                        raise ValueError('legacy runtime consumer imports modern runtime')
                if flavor == 'edition2026' and backend == backends[0]:
                    stable_source = dest
        if not args.legacy_only:
            runtime = manifest['official_runtime']
            selected = json.loads(run('download-runtime-pin', ['go', 'mod', 'download', '-json', runtime['module'] + '@' + runtime['version']]).stdout)
            if selected['Sum'] != runtime['sum'] or selected['GoModSum'] != runtime['go_mod_sum']:
                raise ValueError('official runtime checksum differs from release contract')
        if args.probe_stable and stable_source:
            control = out / 'stable-reference'
            shutil.copytree(stable_source, control)
            runtime = manifest['official_runtime']
            run('stable-override', ['go', 'mod', 'edit', '-replace=' + runtime['module'] + '=' + runtime['module'] + '@' + runtime['stable_reference']], control)
            result = run('stable-metadata-control', ['go', 'test', '-mod=mod', '-count=1', '-run=TestEdition2026Consumer', './...'], control, allow_failure=True)
            gap = runtime['stable_reference_gap']
            if result.returncode != 1 or gap['diagnostic'] not in result.stdout + result.stderr:
                raise ValueError('stable reference gap changed; review dependency policy and probe result')
            report['stable_reference_control'] = {'version': runtime['stable_reference'], 'capability_gap': gap['kind'], 'expected_diagnostic_reproduced': True}
        report['status'] = 'passed'
    except Exception as error:
        report['status'], report['error'] = 'failed', str(error)
        raise
    finally:
        save()
    print('Remote consumer qualification passed:', len(report['consumer_runs']), 'consumer runs')


if __name__ == '__main__':
    main()
