# Proposed root v1.4.0 with existing reflect v0.1.0-rc.2

This is a publication draft, not an announcement that v1.4.0 exists. S04
prepares qualification and a root-only tag plan; no tag or Release is created.
The final publication plan supplies the exact reviewed root commit and its
remote module checksums. Publish only after all current-head reviews and CI
succeed, the commit is contained in baseline/v1.3.2, and separate publication
authorization is obtained. After tagging, rerun published-version installation
and the real-project gate before announcing availability.

## Versions and support

- Root runtime and five generators: proposed `v1.4.0`, canonical module
  `github.com/gogo/protobuf`, minimum Go 1.15, no official V2 dependency.
- Optional modern module: reuse the already published `reflect/v0.1.0-rc.2`,
  canonical module `github.com/gogo/protobuf/reflect`, minimum Go 1.23.
  Its original source remains `3af6d45fd9dbe0ff1249afa246a44caf995b29a1`.
  It retains minimum root `v1.4.0-rc.2`; the proposed stable root satisfies it.
  The remote module zip sum is
  `h1:L1WsD3DuZ/mO7X5G8y2IkSVLuXSNP9gkduYcwIawUSg=` and go.mod sum is
  `h1:QH+77kHBTGQyhwF0lJjcePKNNHD1CcuySLbuRU+ZtLs=`.
- Modern runtime pin: `google.golang.org/protobuf v1.36.13-0.20261007182434-0c373ff94f29`, with exact checksums in
  `manifest.json`. The modern module remains a prerelease. This root promotion
  neither republishes it nor changes its dependency/support policy.

Canonical imports remain unchanged. Applications select remote fork module
replacements and retain go.sum, as in INSTALL.md; once the root tag is verified,
use v1.4.0 for the root require/replacement and keep the reflect RC2 replacement.
Generator installation uses the canonical unversioned package from that tools
module, not an unsupported versioned install against a replacement directive.

## Qualified scope and boundaries

Root v1.4.0 includes the additive compatibility and generator work documented
in the published RC2 notes and the later maintenance branch. Existing projects
retain the old default API and layout. Modern OPEN reflection, HYBRID and OPAQUE
are opt-in and require the separately qualified RC module/runtime combination.
OPAQUE and HYBRID protoopaque builds retain the published ownership/storage
contract; no memory-saving or lazy-decoding claim is added.

The independent combination must pass ten Go1.15 legacy cases and 105 modern
cases on each of Go1.23 and Go1.27, plus the exact stable-runtime negative
control, full compatibility/conformance CI and the four pinned real-project
matrices. The reports, rather than this draft, establish the final tested root
revision and remote checksums. The selected packages include Prometheus,
Kubernetes API/apimachinery, BuildKit and CockroachDB errors; this is not a
whole-product certification.

Legacy MessageSet remains supported. Modern live reflection rejects MessageSet;
non-native custom representations remain outside that live-storage contract.
See COMPATIBILITY.md and release/SUPPORT.md for the complete boundaries and the
reason the modern module stays RC. Existing RC1 and RC2 tags remain immutable.
