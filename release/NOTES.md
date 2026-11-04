# Published v1.4.0-rc.2 and reflect/v0.1.0-rc.2

Published on 2026-11-03 from qualified commit
`3af6d45fd9dbe0ff1249afa246a44caf995b29a1`:
[root RC2](https://github.com/carter-ya/protobuf/releases/tag/v1.4.0-rc.2) and
[reflect RC2](https://github.com/carter-ya/protobuf/releases/tag/reflect/v0.1.0-rc.2).
Both are prereleases, and RC1 tags remain unchanged. The root module keeps
`github.com/gogo/protobuf`; the optional module keeps its `/reflect` suffix.
Use the published fork replacements in [INSTALL.md](INSTALL.md).

## Root runtime and generators

- Preserve old default generated APIs/imports and Go 1.15 runtime support.
- Support proto3 optional and qualified Editions 2023, 2024 and 2026 core
  behavior across gogo, gogofast, gogofaster, gogoslick and gogotypes.
- Offer explicit native ProtoReflect, HYBRID and OPAQUE generation. HYBRID adds setters,
  presence/clear methods, oneof cases and builders while retaining exported
  fields. New API name collisions receive diagnostics; generated private codec
  names track the allocated public accessor/builder names. OPAQUE and the optional module require the version-18 runtime guard.
- Keep historical JSON/text entrypoints; standard behavior is selected explicitly.
  Legacy proto2 closed-enum regeneration and final required-field merge behavior
  also have explicit policies; frozen bindings retain their historical contracts.

## Optional native reflection

- Require Go 1.23 and the manifest's exact official Go runtime snapshot.
- Expose actual native field/list/map/oneof/unknown storage through live reflection.
  Native generated ProtoReflect preserves Go type identity through New, Interface,
  Wrap and Clone; frozen supported bindings can use Wrap.
- Support the qualified closed-enum, required, group and map-backed extension
  profile, including nested resolver and extension merge behavior.
- Ship option-only metadata without V2 global registration. Use an explicit
  `NewRegistry` for Any payloads and Edition 2026 custom enum JSON names.

Opting in may change convenience-generator defaults to pointer/unknown storage.
Review the regenerated diff. Explicit custom/std/embedded mappings, flattened
message values, missing unknown storage, byte-backed extensions and MessageSet
are rejected for the live profile. OPAQUE and the HYBRID `protoopaque` alternate build are qualified in this
published RC2 pair. Proto fields are private; accessors, presence, builders,
legacy/V2 binary, JSONPB/standard JSONPB, expanded Any text and native reflection
are covered. The physical layout retains native pointers and eager decoding.
Legacy MessageSet remains available; only the live profile rejects it. Default old generation remains available.

## Qualification and known differences

The maintained compatibility gates include original/current controls for pinned
Prometheus, Kubernetes, BuildKit and CockroachDB errors, five-backend native
consumers, official reflection/API contracts, race checks, compiler/frontend
inventories and separate stable/preview matrices. Selected-package results do
not claim whole-product cluster/daemon/system coverage or every past release.

Native proto3 all-types has 1,930 C++ differential cases: 1,919 pass, and eleven
codec differences are shared with the pinned official Go control, with no extra
native failure or skip. See `compat/native-conformance/official-failures.json`.
These cases are retained as individual limitations, not reported as passing
conformance. The legacy/standard runtime suites have separate inventories.

The exact source passed four clean reviews with no unresolved threads and all
236 PR checks. After atomic tag publication, both tag pairs passed Go 1.15's
ten legacy cases and Go 1.23/1.27's 105 modern cases, including explicit OPAQUE
and both HYBRID build modes, in fresh external module caches. Full tag origins,
module checksums and dependency graphs were verified. All 62 tag checks passed:
[root installation](https://github.com/carter-ya/protobuf/actions/runs/41562661542),
[reflect installation](https://github.com/carter-ya/protobuf/actions/runs/41562661362),
[root compatibility](https://github.com/carter-ya/protobuf/actions/runs/41562661505),
[reflect compatibility](https://github.com/carter-ya/protobuf/actions/runs/41562661363).

These results establish the published RC2 scope; later maintenance commits
require their own current-head qualification. They do not imply support for
every Protobuf feature, representation, or deployed downstream product.
