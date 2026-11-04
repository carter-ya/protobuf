"""Required remote consumption profiles for the next paired candidate."""
BACKENDS = ('gogo', 'gogofast', 'gogofaster', 'gogoslick', 'gogotypes')
FLAVORS = ('legacy', 'open', 'hybrid', 'opaque', 'edition2026')
PROFILE = 'native-opaque-v1'


def variants(legacy, flavor):
    cases = [('default', []), ('purego', ['-tags=purego'])]
    if not legacy:
        cases.append(('race', ['-race']))
        if flavor in ['hybrid', 'edition2026']:
            cases += [('protoopaque', ['-tags=protoopaque']),
                      ('protoopaque-purego', ['-tags=protoopaque,purego']),
                      ('protoopaque-race', ['-tags=protoopaque', '-race'])]
    return cases


def expected(legacy):
    return {(b, f, v) for b in BACKENDS for f in (['legacy'] if legacy else FLAVORS)
            for v, _ in variants(legacy, f)}
