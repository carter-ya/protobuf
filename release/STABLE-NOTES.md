# Prepared root v1.4.1 patch with existing reflect RC2

This candidate fixes two confirmed runtime boundaries from the master review:
optional standard mappings preserve unknown wrong-wire occurrences, and
standard text maps initialize an omitted closed-enum value from its schema
default. Packed closed-enum decoding keeps the remaining input separately to
avoid quadratic copying; omitted map message values enforce required fields. Targeted regressions reproduce both failures before the fix. Final
publication requires the reviewed source, all CI and actual tag installation.
The existing root v1.4.0 tag and every RC tag remain immutable; reflect RC2 keeps
its original source, Go1.23 floor and fixed official-runtime dependency.

The already verified v1.4.0 installation record below is historical evidence,
not evidence that the v1.4.1 tag has been published.

# Root v1.4.0 stable with existing reflect v0.1.0-rc.2

Root v1.4.0 identifies `057d06490a721bea0fceb39c5c088bc1ef3085e8`. The existing optional reflect
RC2 remains at `3af6d45fd9dbe0ff1249afa246a44caf995b29a1`. These tags deliberately have
separate sources. The root stable tag passed published-version cold-cache
installation on Go1.15, Go1.23 and Go1.27 with the unchanged optional RC module.
See [installation](INSTALL.md) and the per-component identities in manifest.json.

## Versions and support

- Root runtime and five generators: stable `v1.4.0`, canonical module
  `github.com/gogo/protobuf`, minimum Go 1.15, no official V2 dependency.
  Remote zip sum: `h1:BSo4PqFoTb+H4vFKDz9/+kD0Xl4g60KGDWGuo2Vy46o=`;
  go.mod sum: `h1:P1XiOD3dCwIKUDQYPy72D8LYyHL2YPYrpS2s69NZV8Q=`.
- Optional modern module: reuse the already published `reflect/v0.1.0-rc.2`,
  canonical module `github.com/gogo/protobuf/reflect`, minimum Go 1.23.
  Its original source remains `3af6d45fd9dbe0ff1249afa246a44caf995b29a1`.
  It retains minimum root `v1.4.0-rc.2`; the stable root satisfies it.
  The remote module zip sum is
  `h1:L1WsD3DuZ/mO7X5G8y2IkSVLuXSNP9gkduYcwIawUSg=` and go.mod sum is
  `h1:QH+77kHBTGQyhwF0lJjcePKNNHD1CcuySLbuRU+ZtLs=`.
- Modern runtime pin: `google.golang.org/protobuf v1.36.13-0.20261007182434-0c373ff94f29`, with exact checksums in
  `manifest.json`. The modern module remains a prerelease. This root promotion
  neither republishes it nor changes its dependency/support policy.

Canonical imports remain unchanged. Applications select remote fork module
replacements and retain go.sum, as in INSTALL.md; use v1.4.0 for the root
require/replacement and keep the reflect RC2 replacement.
Generator installation uses the canonical unversioned package from that tools
module, not an unsupported versioned install against a replacement directive.

## Qualified scope and boundaries

Root v1.4.0 includes the additive compatibility and generator work documented
in the published RC2 notes and the later maintenance branch. Existing projects
retain the old default API and layout. Modern OPEN reflection, HYBRID and OPAQUE
are opt-in and require the separately qualified RC module/runtime combination.
OPAQUE and HYBRID protoopaque builds retain the published ownership/storage
contract; no memory-saving or lazy-decoding claim is added.

The independent combination passed ten Go1.15 legacy cases and 105 modern
cases on each of Go1.23 and Go1.27, including the exact stable-runtime negative
control. Full compatibility/conformance CI and the four pinned real-project
matrices are required for promotion; published-version CI reports record the
actual versions, source revisions and checksums. The selected packages include
Prometheus,
Kubernetes API/apimachinery, BuildKit and CockroachDB errors; this is not a
whole-product certification.

Legacy MessageSet remains supported. Modern live reflection rejects MessageSet;
non-native custom representations remain outside that live-storage contract.
See COMPATIBILITY.md and release/SUPPORT.md for the complete boundaries and the
reason the modern module stays RC. Existing RC1 and RC2 tags remain immutable.

Actual tag installation evidence: [Go1.15/1.23/1.27 cold-cache consumer CI](https://github.com/carter-ya/protobuf/actions/runs/41800835625).

## Optional-module source fixes after RC2

This source also fixes quadratic unknown-buffer allocation paths and repeated
whole-message validation during closed-enum occurrence decoding in the optional
reflection decoder. The unchanged published reflect RC2 does not
contain those fixes. They require a separately qualified optional-module
release; the root stable tag does not republish the nested module. See the
known source-fix boundary in reflect/README.md before using RC2 on untrusted
wire input with many occurrences or large preexisting unknown buffers.
