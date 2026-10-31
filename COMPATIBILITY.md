# Compatibility-first maintenance

This fork starts at gogo/protobuf v1.3.2 (`b03c65ea`). The module and generated
imports remain `github.com/gogo/protobuf`. Existing generated files remain usable
without regeneration. The runtime has no new module dependencies and retains
`go 1.15`; the isolated official-oracle test module requires Go 1.23.

| Input | Supported output | Runtime guard |
| --- | --- | --- |
| Existing proto2 / proto3 | Original API, field layout, tags and behavior | Version3 |
| proto3 optional | gogo, gogofast, gogofaster, gogoslick, gogotypes | Version4 |
| Editions 2023 / 2024 | Five gogo-runtime backends, historical Open Struct API | Version5 reflection / Version9 fast |
| Edition 2026 | Same backends, custom JSON enum spellings in jsonpb | Version6 reflection / Version9 fast |

New optional fields preserve absent vs explicit zero values. Synthetic oneofs
remain in descriptors but do not appear as public Go oneof wrappers. Explicit
presence also distinguishes nil bytes from present empty bytes in binary, JSON,
text, clone, merge and equality. New optional strings validate UTF-8 in both fast
and reflection paths. Historical fast-string behavior is unchanged.

Editions use a private resolved feature model. Raw syntax, editions, features and
options remain in emitted descriptors; source-retained options are omitted by
protoc. Enum openness comes from the enum definition, including imported enums.
Closed-enum unknown values are retained, including packed values, map entries,
oneofs and registered extensions. Extensions inherit their declaration scope.
DELIMITED messages, required fields and per-field UTF-8/repeated encoding are
supported. Option-only imports do not introduce runtime Go imports.

Edition 2026 JSON names support aliases, empty strings and escaped/Unicode
spellings. Proto names remain accepted in JSON and unchanged in text format.
The ordinary enum name maps and `String` methods retain proto names.

## Deliberate generation errors

- Optional requires the gogo runtime. `nullable=false`, non-bytes customtype,
  message cast, embedded, face and union combinations are rejected.
  Scalar/enum/bytes casttype mappings with matching underlying Go types are
  supported by all five optional backends and require runtime Version10.
  Custom bytes and standard WKT mappings require runtime Version11; mapped
  values retain representation-specific range and metadata limits. Explicit
  unsafe codecs and conflicting mappings are rejected.
  See `compat/optional/README.md` for the exact presence contract.
- Editions permit the tested fast codecs and auxiliary/naming/tag allowlist.
  Custom field representations, unsafe codecs and other unverified extensions
  remain rejected. See `compat/README.md` for the exercised combinations.
- Gogo preserves its Open Struct API in every edition. Explicit API_OPEN and
  keeping enum prefixes are accepted. HYBRID/OPAQUE, prefix stripping and other
  unimplemented explicit Go feature requests are rejected.
- Unvalidated auxiliary plugins do not inherit Editions support.
- Unsupported/future editions fail instead of using the latest known defaults.

These are draft maintenance changes, not a claim that the historical
conformance runner proves current full Protobuf conformance. Tests cover the
implemented feature contracts and independent interoperability paths.

## Verification

`compat/check.sh` compiles frozen v1.3.2 consumers, compares regenerated Go syntax
and decoded descriptors, and runs the full historical suite in a disposable
GOPATH-shaped copy. Formatting/comments and gzip representation may differ by
Go version; declarations, tags, bodies and raw descriptor data must match.

`compat/check-optional.sh` and `compat/check-entrypoints.sh` test optional
presence and generator combinations. `compat/check-features.sh` reproduces the
pinned protoc 36.2 defaults and validates feature inheritance and diagnostics.
`compat/check-editions.sh` tests production generation, raw descriptors, option
retention, runtime guards, and binary/text interoperability with protoc 36.2.
The nested `compat/oracle` module pins official Go commit
`0c373ff94f296da8645fbc892f211083b695709e` for two-way 2026 JSON checks.
`compat/check-gostring.sh` covers the separate pre-existing nil timestamp fix.

CI covers Go 1.23/1.27 with protoc 3.21.8/36.2 and a Go 1.15.15 runtime/compiler
job with protoc 36.2. The modern official oracle runs only on Go 1.23+.

`compat/check-downstream.sh` pins etcd raft v3.5.0 and CockroachDB errors v1.8.1
(their manifests declare gogo v1.3.2 and v1.3.1 respectively). It tests frozen
bindings, binary/JSON/Any and gRPC error roundtrips, raft's full package suite,
and CockroachDB errbase/extgrpc tests. It then regenerates raft with the original
and candidate gofast generators under the same protoc, compares code and raw
descriptors, and repeats raft's suite with regenerated bindings and no business
source edits. These dependencies are isolated from the root module.

Etcd's frozen bindings use the golang runtime despite their gogo generator.
Regeneration preserves that gofast choice. The smoke test uses integer enum JSON;
looking up its named enum JSON through gogo's separate registry already fails
with untouched v1.3.2 and is not broadened as part of this upgrade.

## Explicit standard JSON entrypoint

`jsonpb/standard` offers whole-document parsing, Unicode validation, presence
preservation and deterministic duplicate/oneof rejection. Existing `jsonpb`
configuration layouts and default behavior are unchanged. Modern conformance
runs both entrypoints independently and records remaining limitations; see
`jsonpb/standard/README.md` and `compat/conformance/README.md`.

Standard JSON also supports full-range Duration, range-checked Timestamp,
FieldMask strings, empty Any and Any-wrapped special types. It rejects
non-finite Value numbers. Legacy jsonpb WKT representations and tolerance
remain unchanged. These paths are checked against both the pinned C++ suites
and official Go implementation.

## Editions metadata in standard types

`Syntax_SYNTAX_EDITIONS` is value 2. Type, Enum, Api and Method gain the ordinary
string `Edition` field with their official field numbers (7, 6, 8 and 8).
Method's syntax and edition metadata are deprecated in favor of the parent Api,
matching upstream. Existing field numbers, defaults, imports and runtime guard
remain unchanged. These additions do change the public WKT struct shapes;
consumers using positional literals or depending on in-memory layout must adapt.
Frozen original bindings remain usable and retain new metadata as unknown bytes.
`compat/check-wkt-metadata.sh` verifies the exact permitted schema delta, generated
snapshots and original consumers; the isolated official Go oracle checks both
binary and JSON representations.

Messages with more than 64 required fields now use an additional validation
bitmap. Files containing such messages require runtime Version7, including
proto2 files; smaller legacy messages keep their original guard and allocation
path. The boundary consumer checks 63/64/65/128/129 fields, sparse tags, absent
fields, explicit zeroes, nested parsing and Merge.

Editions now support an explicit allowlist of getter/naming/tag options,
Equal/Compare, Populate, Stringer and GoString. Empty bytes retain presence in
generated equality/ordering. Unknown-field storage cannot be disabled. Custom
representations, unsafe codecs, face/union and unverified options still fail
with a scoped diagnostic. Auxiliary plugins advertise Editions only through an
explicit capability; unrecognized plugins do not inherit it.

## Modern Go conversion experiment

`compat/modern` evaluates detached V2 dynamic snapshots with explicit replacement,
local registries, protojson, Any, extensions and five-backend optional mappings.
It also exercises actual frozen raft and CockroachDB errors bindings. It is an
isolated Go 1.23 test module, not a generated ProtoReflect API or a released
adapter. Data that cannot be retained on replacement is rejected before mutation.
The pinned official dynamic runtime's closed-enum unknown interpretation and
MessageSet restrictions are recorded in `compat/modern/README.md`; no transparent
adaptation claim is made for those cases.

## Explicit standard text entrypoint

`textpb/standard` adds standard floating-point syntax, negative-zero output,
closed-enum and UTF-8 policy checks while preserving the historical text APIs
and TextMarshaler layout. The first implementation passes 125 formerly failing
standard text cases. Unicode scalar validation and descriptor-backed reserved-name
skipping remove 72 more. Any URL syntax removes another 8 and schema-based
group names resolve the last 18. All 909 pinned Protobuf 36.2 text cases pass
through the standard entrypoint without skips; the legacy baseline stays separate. Unknown fields
can be omitted or printed using the explicit configuration. It is not a
full text-conformance claim. See `textpb/standard/README.md`.

## Concurrent read-only encoding

JSON encoding checks time, duration and custom JSON interfaces by reflected type
before materializing values. Ordinary messages are no longer copied while binary
sizing updates internal caches. The runtime race gate covers proto, legacy and
standard JSON, and standard text with default and purego builds. This contract
assumes application fields are not being mutated concurrently.

## MessageSet input compatibility

MessageSet accepts both item groups and ordinary length-delimited extension
fields. Mixed encodings retain wire order and output remains in MessageSet
format. Merging after SetExtension or GetExtension retains decoded payloads.
Unknown extension bodies remain opaque. The explicit V2 snapshot experiment
continues to reject MessageSet because its target runtime does not support it.
