#!/usr/bin/env python3
"""Qualify pinned real projects with original and maintained gogo implementations."""
import argparse
import datetime
import hashlib
import io
import json
import os
import pathlib
import shutil
import subprocess
import tarfile

from published import PublishedPair, environment

ROOT = pathlib.Path(__file__).resolve().parents[3]
HERE = pathlib.Path(__file__).resolve().parent


def output(command, cwd=ROOT, env=None):
    return subprocess.check_output([str(x) for x in command], cwd=cwd, env=env, text=True).strip()


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('output', type=pathlib.Path)
    parser.add_argument('--profile', action='append')
    parser.add_argument('--frozen-only', action='store_true')
    selection = parser.add_mutually_exclusive_group()
    selection.add_argument('--published', action='store_true', help='test the recorded published pair from remote modules in a cold cache')
    selection.add_argument('--candidate-root', help='qualify this exact public stable-root candidate with the frozen published reflect module')
    parser.add_argument('--candidate-repository', help='public root repository for candidate commits; frozen modules keep their recorded publisher')
    args = parser.parse_args()
    if args.candidate_repository and not args.candidate_root:
        parser.error('--candidate-repository requires --candidate-root')
    remote_pair = bool(args.published or args.candidate_root)
    dest = args.output.resolve()
    if remote_pair and dest.is_relative_to(ROOT):
        parser.error('published qualification requires an external output directory')
    dest.mkdir(parents=True)
    pins = json.loads((HERE / 'pins.json').read_text())
    chosen = [p for p in pins if not args.profile or p['name'] in args.profile]
    if not chosen or (args.profile and set(args.profile) - {p['name'] for p in pins}):
        parser.error('unknown project profile')
    env = environment(dest) if remote_pair else dict(os.environ, GOWORK='off', GOTOOLCHAIN='local')
    report = {'status': 'running', 'time': datetime.datetime.now(datetime.timezone.utc).isoformat(),
              'source_revision': output(['git', 'rev-parse', 'HEAD']),
              'working_tree': output(['git', 'status', '--porcelain']),
              'go': output(['go', 'version'], env=env), 'protoc': output(['protoc', '--version'], env=env),
              'mode': 'candidate-root-published-reflect' if args.candidate_root else 'published-pair' if args.published else 'source-checkout',
              'cold_module_cache': remote_pair,
              'harness_sha256': hashlib.sha256(pathlib.Path(__file__).read_bytes()).hexdigest(),
              'published_helper_sha256': hashlib.sha256((HERE / 'published.py').read_bytes()).hexdigest(),
              'templates_sha256': {p.name: hashlib.sha256(p.read_bytes()).hexdigest() for p in sorted(HERE.glob('*.go.txt'))},
              'pins_sha256': hashlib.sha256((HERE / 'pins.json').read_bytes()).hexdigest(),
              'original_revision': output(['git', 'rev-parse', 'v1.3.2']), 'checks': [], 'projects': []}

    def save():
        (dest / 'qualification.json').write_text(json.dumps(report, indent=2) + '\n')

    def run(name, command, cwd=ROOT, extra=None, capture=False):
        log = dest / (name + '.log')
        options = {'cwd': cwd, 'env': dict(env, **(extra or {})), 'text': True}
        if capture:
            result = subprocess.run([str(x) for x in command], stdout=subprocess.PIPE, stderr=subprocess.PIPE, **options)
            log.write_text(result.stdout + result.stderr)
        else:
            with log.open('w') as stream:
                result = subprocess.run([str(x) for x in command], stdout=stream, stderr=subprocess.STDOUT, **options)
        report['checks'].append({'name': name, 'command': [str(x) for x in command], 'cwd': str(cwd),
                                 'passed': result.returncode == 0, 'log': log.name})
        save()
        if result.returncode:
            raise RuntimeError(name + ' failed; inspect ' + str(log))
        print(name, 'passed', flush=True)
        return result

    save()
    try:
        published = PublishedPair(dest, run, report, args.candidate_root, args.candidate_repository) if remote_pair else None
        if published and report['protoc'] != 'libprotoc ' + published.manifest['compiler']:
            raise ValueError('published downstream matrix requires its pinned compiler')
        includes = published.root if published else ROOT
        original = dest / 'original-gogo'
        original.mkdir()
        archive = subprocess.check_output(['git', 'archive', 'v1.3.2'], cwd=ROOT)
        with tarfile.open(fileobj=io.BytesIO(archive)) as source:
            source.extractall(original, filter='data')
        # Normalize development-only dependencies for the current Go compiler.
        # All original production Go sources remain from the immutable v1.3.2 tag.
        for name in ['go.mod', 'go.sum']:
            shutil.copyfile(ROOT / name, original / name)
        drivers = {}
        for profile in chosen:
            name = profile['name']
            modules = []
            for index, pin in enumerate(profile['modules']):
                command = ['go', 'mod', 'download', '-json', pin['Path'] + '@' + pin['Version']]
                raw = run(name + '-download-project-' + str(index), command, capture=True).stdout if published else output(command, env=env)
                data = json.loads(raw)
                for field in ['Path', 'Version', 'Sum', 'GoModSum']:
                    if data.get(field) != pin[field]:
                        raise RuntimeError('module checksum or version changed: ' + pin['Path'])
                if 'github.com/gogo/protobuf v1.3.2' not in pathlib.Path(data['GoMod']).read_text():
                    raise RuntimeError('project no longer declares the qualified gogo dependency')
                modules.append(data)
            entry = {'name': name, 'modules': [{k: m[k] for k in ['Path', 'Version', 'Sum', 'GoModSum']} for m in modules], 'modes': []}
            report['projects'].append(entry)
            generation = ['frozen'] + ([] if args.frozen_only or not profile['regenerate'] else ['regenerated'])
            comparisons = {}
            generated_outputs = {}
            consumers = {}

            def semantics(record):
                return {name: {k: v for k, v in value.items() if k != "wire"} for name, value in record.items()}
            for mode in generation:
                for runtime, implementation in [('original', original), ('fork', includes)]:
                    label = name + '-' + mode + '-' + runtime
                    consumer = dest / label
                    consumer.mkdir()
                    (consumer / 'common_test.go').write_text((HERE / 'common_test.go.txt').read_text())
                    (consumer / 'project_test.go').write_text((HERE / (name + '_test.go.txt')).read_text())
                    if name != 'kubernetes':
                        (consumer / 'json_test.go').write_text((HERE / 'protojson_test.go.txt').read_text())
                    root_version = published.versions['root'] if published and runtime == 'fork' else 'v1.3.2'
                    mod = 'module example.com/gogo-project-compat\n\ngo 1.26.0\n\nrequire (\n github.com/gogo/protobuf ' + root_version + '\n'
                    mod += ''.join(' ' + m['Path'] + ' ' + m['Version'] + '\n' for m in modules) + ')\n'
                    replacement = published.replacement('root') if published and runtime == 'fork' else str(implementation)
                    mod += 'replace github.com/gogo/protobuf => ' + replacement + '\n'
                    local_project = {}
                    if mode == 'regenerated':
                        source = dest / (label + '-source')
                        shutil.copytree(modules[0]['Dir'], source)
                        for path in [source] + list(source.rglob('*')):
                            if not path.is_symlink():
                                path.chmod(path.stat().st_mode | 0o200)
                        spec = profile['regenerate']
                        key = (runtime, spec['backend'])
                        if key not in drivers:
                            if published and runtime == 'fork':
                                driver = published.generator(spec['backend'], report)
                            else:
                                driver = dest / (runtime + '-' + spec['backend'])
                                run(runtime + '-build-' + spec['backend'], ['go', 'build', '-o', driver, './protoc-gen-' + spec['backend']], implementation)
                            drivers[key] = driver
                        include = dest / 'include/github.com/gogo'
                        include.mkdir(parents=True, exist_ok=True)
                        alias = include / 'protobuf'
                        if not alias.exists():
                            alias.symlink_to(includes, target_is_directory=True)
                        directory = source / spec['directory']
                        params = ['paths=source_relative'] + spec['options']
                        groups = {}
                        for file in spec['files']:
                            groups.setdefault(str(pathlib.Path(file).parent), []).append(file)
                        for index, group in enumerate(groups.values()):
                            run(label + '-generate-' + str(index), ['protoc', '-I' + str(directory), '-I' + str(source), '-I' + str(includes),
                                '-I' + str(includes / 'protobuf'), '-I' + str(dest / 'include'),
                                '--plugin=protoc-gen-test=' + str(drivers[key]), '--test_out=' + ','.join(params) + ':' + str(directory)] + group)
                        outputs = [directory / pathlib.Path(p).with_suffix('.pb.go') for p in spec['files']]
                        generated_outputs[runtime] = outputs
                        if runtime == 'fork':
                            for index, (before, after) in enumerate(zip(generated_outputs['original'], outputs)):
                                if published:
                                    run(label + '-compare-' + str(index), ['go', 'run', '-mod=mod', 'github.com/gogo/protobuf/compat/compare-generated', '--', before, after], published.tools)
                                else:
                                    run(label + '-compare-' + str(index), ['go', 'run', './compat/compare-generated', '--', before, after])
                        mod += 'replace ' + modules[0]['Path'] + ' => ' + str(source) + '\n'
                        local_project[modules[0]['Path']] = source
                    if runtime == 'fork':
                        reflect_version = published.versions['reflect'] if published else 'v0.0.0'
                        runtime_version = published.runtime if published else 'v1.36.13-0.20261007182434-0c373ff94f29'
                        mod += 'require (\n github.com/gogo/protobuf/reflect ' + reflect_version + '\n google.golang.org/protobuf ' + runtime_version + '\n)\n'
                        mod += 'replace github.com/gogo/protobuf/reflect => ' + (published.replacement('reflect') if published else str(ROOT / 'reflect')) + '\n'
                        (consumer / 'reflection_test.go').write_text((HERE / 'reflection_test.go.txt').read_text())
                        if name == 'prometheus':
                            (consumer / 'reflection_prometheus_test.go').write_text((HERE / 'reflection_prometheus_test.go.txt').read_text())
                    (consumer / 'go.mod').write_text(mod)
                    if not published:
                        shutil.copyfile(ROOT / 'go.sum', consumer / 'go.sum')
                    records = []
                    consumers[(mode, runtime)] = consumer
                    for variant, flags in [('default', []), ('purego', ['-tags=purego']), ('race', ['-race'])]:
                        path = consumer / (variant + '.json')
                        run(label + '-' + variant, ['go', 'test', '-mod=mod', '-count=1', '-timeout=5m'] + flags + ['./...'] + profile['packages'],
                            consumer, dict({'PROJECT_COMPAT_REPORT': str(path), 'PROJECT_COMPAT_RUNTIME': runtime, 'PROJECT_COMPAT_VARIANT': variant}, **({'PROJECT_COMPAT_PEER_REPORT': str(consumers[(mode, 'original')] / 'default.json')} if runtime == 'fork' else {})))
                        if published:
                            published.graph(label + '-graph-' + variant, consumer, report,
                                            original=original if runtime == 'original' else None, local_project=local_project)
                        records.append(json.loads(path.read_text()))
                    if any(semantics(record) != semantics(records[0]) for record in records[1:]):
                        raise RuntimeError('default/purego/race message semantics differ: ' + label)
                    comparisons[(mode, runtime)] = records[0]
                    entry['modes'].append({'generation': mode, 'runtime': runtime, 'variants': ['default', 'purego', 'race'], 'messages': len(records[0])})
                if semantics(comparisons[(mode, 'original')]) != semantics(comparisons[(mode, 'fork')]):
                    raise RuntimeError('original/fork message size or JSON differ: ' + name + '-' + mode)
                for variant, flags in [('default', []), ('purego', ['-tags=purego']), ('race', ['-race'])]:
                    run(name + '-' + mode + '-original-read-fork-' + variant, ['go', 'test', '-mod=mod', '-count=1', '-run=^TestRealProjectMessages$'] + flags + ['./...'], consumers[(mode, 'original')], {'PROJECT_COMPAT_PEER_REPORT': str(consumers[(mode, 'fork')] / 'default.json')})
        save()
        report['status'] = 'passed'
    except Exception:
        report['status'] = 'failed'
        raise
    finally:
        save()
    print(dest / 'qualification.json')


if __name__ == '__main__':
    main()
