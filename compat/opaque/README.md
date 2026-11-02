# OPAQUE development qualification

The published root v1.4.0-rc.1 / reflect v0.1.0-rc.1 pair does not include this
development work. Explicit OPAQUE generation is being qualified on top of that
pair; do not describe the earlier tags as supporting it.

`prepare.py OUTPUT` uses protoc 36.2 and Go 1.23 or newer. It exercises all five
maintained generators across proto3, proto2 and Editions 2023/2024/2026 in default,
purego and race modes. The proto2 cases include closed enums, registered
extensions, required fields and groups. A second schema checks getter/field and
nested-type/oneof-wrapper name collisions. Reports contain 75 profile runs plus
15 name/default runs. Defaults cover scalar/string/escaped bytes/enum presence
and recursive child/list/map traversal.

Pass `--official-generator PATH` to compare the OPAQUE accessors, getters,
builders, cases and ProtoReflect signatures with the pinned official generator.
External direct-field access must fail compilation. Behavioral checks include
official prototest, live reflection, nil/zero presence, builders and aliasing,
native identity, legacy/V2 binary interoperability, both legacy and standard JSONPB, and binary/JSON/text round
trips. The selected profile retains the existing representation restrictions.

Existing generator defaults remain unchanged. OPAQUE is explicitly selected
with `default_api_level=API_OPAQUE` or the Go `api_level` feature. This layer
does not yet qualify HYBRID's `protoopaque` alternate build; that is a separate
required follow-up. Private storage currently retains the native pointer layout
and eager decoding; this qualification makes no memory-saving or lazy-decoding
performance claim.
