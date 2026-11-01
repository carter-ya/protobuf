# Proposed v1.4.0-rc.1 and reflect/v0.1.0-rc.1

This document is a prepared RC announcement. This preparation does not publish
tags or GitHub Releases; consult remote tags and Releases for publication status.
Both components must come from the same qualified commit. The root module keeps
`github.com/gogo/protobuf`; the optional module keeps its `/reflect` suffix.
Use the fork replacements in [INSTALL.md](INSTALL.md).

## Root runtime and generators

- Preserve old default generated APIs/imports and Go 1.15 runtime support.
- Support proto3 optional and qualified Editions 2023, 2024 and 2026 core
  behavior across gogo, gogofast, gogofaster, gogoslick and gogotypes.
- Offer explicit native ProtoReflect and HYBRID generation. HYBRID adds setters,
  presence/clear methods, oneof cases and builders while retaining exported
  fields. New API name collisions receive diagnostics rather than renaming
  existing fields. Runtime version-17 guards detect an older runtime.
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
are rejected for the live profile. OPAQUE and the `protoopaque` alternate build
are not implemented. Default old generation remains available.

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

The candidate still requires the release-specific cold-cache remote-consumer
report. The versioned post-tag gate and actual release publication are separate
steps. Do not convert code-review/CI success into a claim that a package has
already been published or that all Protobuf features are implemented.
