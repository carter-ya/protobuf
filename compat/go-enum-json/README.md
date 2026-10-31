# Go legacy enum JSON API

The official `features.(pb.go).legacy_unmarshal_json_enum` setting controls the
deprecated enum `UnmarshalJSON` method. All five gogo backends implement file
inheritance, nested enums and enum-level overrides. Fresh Editions output uses
the official false default. Set the feature to true to retain the method when
regenerating an existing Editions schema. Frozen bindings remain usable, and
proto2/proto3 generation keeps its original method policy. No struct layout or
runtime dependency changes are required.

This option affects Go's `encoding/json` method interface. It does not choose
the Protobuf JSON message entrypoint; standard JSON/text still use their explicit
packages. Gogo continues to emit the Open Struct API. HYBRID/OPAQUE and optimization requests remain separately tracked until
implemented. Enum prefix settings have a separate qualified matrix.

Edition 2023 uses ordinary imports for `google/protobuf/go_features.proto`.
Without an explicit `M` mapping, gogo maps that schema's official Go package to
the local `github.com/gogo/protobuf/gofeatures` binding. Edition 2024/2026 can use
`import option`. Explicit import mappings remain authoritative. The typed local
extension and complete raw descriptor avoid modern Go runtime imports and share
one registered extension descriptor with the generator. The feature audit now
checks all four compiled Go feature definitions and enum values against the
compiler's own schema, even where the backend still diagnoses unimplemented
semantics.

`sh compat/check-go-enum-json.sh` runs 90 isolated consumers: five backends,
three Editions, default/true/false file settings, default/purego. It checks the
method set, inherited/overridden settings, named/numeric JSON, invalid names and
int32 overflow without mutation. A shared consumer runs against the pinned
official Go generator: stable v1.36.12 supports Editions through 2024; the
v1.37.0-rc.1 preview additionally exercises 2026. Modern oracle dependencies are
isolated from the Go 1.15 production module. The pre-Editions compiler skips this
feature-specific gate.

The local binding is generated from the pinned compiler's unmodified semantic
schema, with explicit import mappings to gogo packages. Source and compiled
schema metadata are retained; receiving new fields does not automatically enable
new generated APIs. Unknown explicit Go features continue to fail generation.
