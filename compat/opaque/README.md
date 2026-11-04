# Published OPAQUE and protoopaque qualification

The published root `v1.4.0-rc.2` / `reflect/v0.1.0-rc.2` pair includes explicit
OPAQUE and HYBRID `protoopaque` builds at commit
`3af6d45fd9dbe0ff1249afa246a44caf995b29a1`. See the
[published installation guide](../../release/INSTALL.md). RC1 excludes OPAQUE;
both release pairs remain immutable as the qualification harness evolves.

`prepare.py OUTPUT` uses protoc 36.2 and Go 1.23 or newer. It exercises all five
maintained generators across proto3, proto2 and Editions 2023/2024/2026 in default,
purego and race modes. The proto2 cases include closed enums, registered
extensions, required fields and groups. A second schema checks getter/field and
nested-type/oneof-wrapper name collisions. Reports contain 75 profile runs plus
15 name/default runs. Defaults cover scalar/string/escaped bytes/enum presence
and recursive child/list/map traversal, plus legacy/standard expanded Any text.

Pass `--official-generator PATH` to compare the OPAQUE accessors, getters,
builders, cases and ProtoReflect signatures with the pinned official generator.
External direct-field access must fail compilation. Behavioral checks include
official prototest, live reflection, nil/zero presence, builders and aliasing,
native identity, legacy/V2 binary interoperability, both legacy and standard JSONPB, and binary/JSON/text round
trips. The selected profile retains the existing representation restrictions.

Existing generator defaults remain unchanged. OPAQUE is explicitly selected
with `default_api_level=API_OPAQUE` or the Go `api_level` feature. HYBRID also emits a `_protoopaque.pb.go` alternate, selected with the
`protoopaque` build tag. `apilevelMfile.proto=API_OPEN|API_HYBRID|API_OPAQUE`
sets a file default; explicit source features and nested overrides remain
authoritative. Private storage currently retains the native pointer layout
and eager decoding; this qualification makes no memory-saving or lazy-decoding
performance claim.


`alternate.py OUTPUT --official-generator PATH` checks 150 normal/alternate
profile runs, including build-specific annotations and declaration order.
`mixed.py OUTPUT` checks 30 cross-package OPEN/HYBRID/OPAQUE combinations;
`generated_tests.py OUTPUT` checks 30 generated gadget-test builds. Each uses
all five generators with default/purego/race modes. Normal HYBRID tests retain
field access and scalar-pointer sharing; OPAQUE tests reject field access and
match official numeric-scalar builder ownership. RC2's published-mode cold-cache
installation checks include explicit OPAQUE and both HYBRID build modes on
Go1.23/Go1.27; [release notes](../../release/NOTES.md) link the actual tag CI.
