"""Explicit component origins for paired RCs and independent root promotion."""
import base64
import hashlib
import re


def revision(value):
    if not isinstance(value, str) or not re.fullmatch('[0-9a-f]{40}', value):
        raise ValueError('component source must be a full immutable revision')
    return value


def published_revisions(pair):
    # Historical paired records have a common revision. New records retain the
    # root alias for old readers and explicitly name both component origins.
    result = pair.get('revisions', {'root': pair['revision'], 'reflect': pair['revision']})
    if set(result) != {'root', 'reflect'} or result['root'] != pair['revision']:
        raise ValueError('published component revisions differ from the root alias')
    return {name: revision(value) for name, value in result.items()}


def root_stable(manifest):
    if manifest['state'] not in ['release-candidate', 'root-stable-modern-rc']:
        raise ValueError('unknown component release mode')
    return manifest['state'] == 'root-stable-modern-rc'


def component_revisions(manifest, candidate):
    result = {'root': revision(candidate), 'reflect': revision(candidate)}
    if root_stable(manifest):
        if manifest['reflect']['tag'] != manifest['published_pair']['reflect'] or manifest['reflect']['version'] != manifest['published_pair']['modules']['reflect']['Version']:
            raise ValueError('root promotion must reuse the recorded optional module')
        result['reflect'] = published_revisions(manifest['published_pair'])['reflect']
    return result


def frozen_component(manifest, name):
    return root_stable(manifest) and name == 'reflect'


def publish_components(manifest):
    return ['root'] if root_stable(manifest) else ['root', 'reflect']


def component_path(manifest, name, root_repository):
    if frozen_component(manifest, name):
        return manifest['published_pair']['modules'][name]['Path']
    return root_repository + ('/reflect' if name == 'reflect' else '')


def check_frozen_identity(manifest, name, downloaded):
    if frozen_component(manifest, name):
        identity = manifest['published_pair']['modules'][name]
        if any(downloaded.get(key) != identity[key] for key in ['Path', 'Version', 'Sum', 'GoModSum']):
            raise ValueError('reused published component identity changed')


def version_key(value):
    match = re.fullmatch(r'v(\d+)\.(\d+)\.(\d+)(?:-rc\.(\d+))?', value)
    if not match:
        raise ValueError('version outside the qualified stable/RC channels')
    major, minor, patch = map(int, match.group(1, 2, 3))
    return major, minor, patch, int(match.group(4) is None), int(match.group(4) or 0)


def go_mod_sum(raw):
    # Go's dirhash.Hash1 for the single virtual file named "go.mod".
    line = hashlib.sha256(raw).hexdigest() + '  go.mod\n'
    return 'h1:' + base64.b64encode(hashlib.sha256(line.encode()).digest()).decode()
