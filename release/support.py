"""Validate the explicitly selected support line, without upgrading dependencies."""
import json
import pathlib
import re

HERE = pathlib.Path(__file__).resolve().parent


def validate(policy, manifest):
    if policy['schema'] != 1:
        raise ValueError('unknown support policy schema')
    root, modern, pairing = (policy[k] for k in ['root', 'reflect', 'pairing'])
    if root != {'minimum_go': '1.15', 'independent_stable_promotion': True, 'official_v2_dependency': False}:
        raise ValueError('root support policy needs explicit review')
    if modern != {'minimum_go': '1.23', 'channel': 'release-candidate',
                  'runtime_policy': 'pinned-development-runtime-for-rc',
                  'stable_requires_qualified_official_stable_runtime': True,
                  'minimum_go_change': 'explicit-versioned-policy-change'}:
        raise ValueError('modern support line needs explicit review')
    if set(pairing) != {'immutable_published_components', 'per_component_revision_and_checksum_evidence',
                        'cold_cache_and_real_downstream_qualification', 'manual_publication'} or not all(v is True for v in pairing.values()):
        raise ValueError('release pairing safeguards changed')
    if manifest['root']['go'] != root['minimum_go'] or manifest['reflect']['go'] != modern['minimum_go']:
        raise ValueError('manifest minimum Go differs from support policy')
    if not re.fullmatch(r'v0\.\d+\.\d+-rc\.\d+', manifest['reflect']['version']):
        raise ValueError('this modern support line remains prerelease')
    if manifest['official_runtime']['policy'] != modern['runtime_policy']:
        raise ValueError('modern dependency policy differs')
    observation = policy['upstream_observation']
    if not re.fullmatch(r'\d{4}-\d{2}-\d{2}', observation['checked_on']) or len(observation['sources']) < 2:
        raise ValueError('upstream decision lacks dated sources')
    if observation['stable_reference'] != manifest['official_runtime']['stable_reference'] or observation['stable_reference_gap'] != manifest['official_runtime']['stable_reference_gap']['kind']:
        raise ValueError('stable control and support decision differ')
    if observation['next_release_minimum_go'] != '1.26':
        raise ValueError('upstream Go observation needs review')


if __name__ == '__main__':
    validate(json.loads((HERE / 'support-policy.json').read_text()), json.loads((HERE / 'manifest.json').read_text()))
    print('Root Go1.15 and optional Go1.23 RC support lines remain explicit.')
