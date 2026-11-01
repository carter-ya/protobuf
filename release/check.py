#!/usr/bin/env python3
"""Validate the paired release contract without publishing any artifacts."""
import json
import pathlib
import re

ROOT = pathlib.Path(__file__).resolve().parents[1]


def directive(text, name):
    match = re.search(r'^' + name + r'\s+(\S+)\s*$', text, re.M)
    if not match:
        raise ValueError('missing ' + name + ' directive')
    return match.group(1)


def require_version(text, module):
    match = re.search(r'^\s*(?:require\s+)?' + re.escape(module) + r'\s+(v\S+)(?:\s+//[^\n]*)?\s*$', text, re.M)
    if not match:
        raise ValueError('missing requirement: ' + module)
    return match.group(1)


def validate(manifest, root_mod, reflect_mod, sums):
    if manifest['schema'] != 1 or manifest['state'] != 'release-candidate' or manifest['publication'] != 'manual':
        raise ValueError('unknown release contract state')
    root, modern, runtime = (manifest[k] for k in ['root', 'reflect', 'official_runtime'])
    if root['module'] != 'github.com/gogo/protobuf' or modern['module'] != root['module'] + '/reflect':
        raise ValueError('canonical module paths changed')
    if directive(root_mod, 'module') != root['module'] or directive(reflect_mod, 'module') != modern['module']:
        raise ValueError('module declaration differs from release manifest')
    if directive(root_mod, 'go') != root['go'] or root['go'] != '1.15':
        raise ValueError('legacy Go compatibility changed')
    if directive(reflect_mod, 'go') != modern['go'] or modern['go'] != '1.23':
        raise ValueError('modern Go compatibility changed')
    if not re.fullmatch(r'v1\.\d+\.\d+-rc\.\d+', root['version']):
        raise ValueError('root candidate version is invalid')
    if not re.fullmatch(r'v0\.\d+\.\d+-rc\.\d+', modern['version']):
        raise ValueError('reflect candidate version is invalid')
    if root['tag'] != root['version'] or modern['tag'] != 'reflect/' + modern['version']:
        raise ValueError('submodule tag/version pairing changed')
    if modern['minimum_root'] != root['version'] or require_version(reflect_mod, root['module']) != root['version']:
        raise ValueError('minimum root must match paired candidate')
    if runtime['stable_reference_gap'] != {'kind': 'edition-2026-runtime-support', 'diagnostic': 'edition EDITION_2026 not yet supported by the Go Protobuf runtime'}:
        raise ValueError('stable reference capability gap needs explicit review')
    if runtime['policy'] != 'pinned-development-runtime-for-rc':
        raise ValueError('runtime policy needs explicit review')
    if require_version(reflect_mod, runtime['module']) != runtime['version']:
        raise ValueError('official runtime pin drifted')
    if not re.fullmatch(r'v\d+\.\d+\.\d+-0\.\d{14}-[0-9a-f]{12}', runtime['version']):
        raise ValueError('expected immutable development pseudo-version')
    if not runtime['version'].endswith('-' + runtime['commit_prefix']):
        raise ValueError('runtime revision differs from pseudo-version')
    for suffix, key in [('', 'sum'), ('/go.mod', 'go_mod_sum')]:
        expected = runtime['module'] + ' ' + runtime['version'] + suffix + ' ' + runtime[key]
        if expected not in sums.splitlines():
            raise ValueError('runtime checksum drifted: ' + key)
    if 'google.golang.org/protobuf' in root_mod:
        raise ValueError('legacy root acquired a V2 dependency')


def main():
    manifest = json.loads((ROOT / 'release/manifest.json').read_text())
    validate(manifest, (ROOT / 'go.mod').read_text(), (ROOT / 'reflect/go.mod').read_text(),
             (ROOT / 'reflect/go.sum').read_text())
    for name in ['INSTALL.md', 'NOTES.md', 'README.md']:
        if not (ROOT / 'release' / name).is_file():
            raise ValueError('missing release deliverable: ' + name)
    print('Paired RC versions, minimum toolchains/root, dependency policy and checksums match.')


if __name__ == '__main__':
    main()
