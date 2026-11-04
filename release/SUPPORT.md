# Root and optional-module support decision

Decision date: 2026-11-03. The machine-readable policy is
[`support-policy.json`](support-policy.json); `python3 release/support.py` checks
that the release contract has not silently changed the selected support line.

| Component | Compatibility floor | Current release decision |
| --- | --- | --- |
| Root runtime and generators | Go 1.15 | Eligible for independent stable promotion after qualification |
| Optional live reflection | Go 1.23 | Keep the RC line and its immutable official-runtime snapshot |

These are this fork's tested compatibility floors. They are not a claim that
upstream Go or official Protobuf supports those old toolchains indefinitely.
The root module retains canonical imports and has no official V2 dependency.
Applications selecting native modern APIs also select the separately qualified
optional module and its declared runtime dependency.

## Why the modern module remains RC

The checked official stable reference is
[v1.36.12](https://github.com/protocolbuffers/protobuf-go/releases/tag/v1.36.12).
The reproducible release control rejects the required Edition 2026 metadata
with the exact diagnostic retained in `manifest.json`. The qualified immutable
snapshot and checksums remain in that manifest and `reflect/go.sum`; switching
to the lower stable reference must not remove already supported semantics.

The [official 2026-10-16 policy](https://protobuf.dev/news/2026-10-16/) raises the
next release's Go requirement to 1.26. The
[v1.37.0-rc.1 module](https://raw.githubusercontent.com/protocolbuffers/protobuf-go/v1.37.0-rc.1/go.mod)
already declares `go 1.26`. This dated observation does not automatically change
our Go 1.23 line or make a preview a stable dependency.

A future modern stable line requires a suitable official stable runtime,
verification of Edition 2026/enum metadata and full codec/reflection/downstream
qualification, plus an explicit versioned decision about any higher Go floor.
The existing RC tags and their dependencies remain immutable. They are not
silently upgraded, reclassified as stable or kept alive by dropping features.

## Independent root promotion

A root stable release may continue to pair with the existing reflect RC2.
Its notes must state both exact versions, each source revision and checksum,
the modern module's RC status and its runtime pin. A new root commit does not
move or republish the existing optional-module tag.

The pairing must be qualified using fresh remote modules: minimum Go legacy
consumers, the complete native OPEN/HYBRID/OPAQUE/Edition 2026 matrix, and pinned
real downstream packages with original controls and supported regeneration.
The exact root head must have a clean review, zero unresolved threads and all
applicable CI successful. Source-checkout success alone does not establish a
versioned or cross-revision package pairing.

S04 prepared the root stable contract, origin-aware evidence and reviewable
publication plan. The separately authorized root v1.4.1 tag has now passed
published-version installation with the existing reflect RC2. Future stable
publication remains an explicit maintainer action with the same verification. This decision leaves MessageSet and representation
boundaries unchanged and makes no new OPAQUE memory/lazy-decoding claim.
