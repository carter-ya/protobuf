# Compatibility-first maintenance

This fork starts at gogo/protobuf v1.3.2 (`b03c65ea`). The module and generated
imports remain `github.com/gogo/protobuf`. Existing generated files remain usable
without regeneration. The runtime has no new module dependencies and retains
`go 1.15`; the isolated official-oracle test module requires Go 1.23.

| Input | Supported output | Runtime guard |
| --- | --- | --- |
| Existing proto2 / proto3 | Original API, field layout, tags and behavior | Version3 |
| proto3 optional | gogo, gogofast, gogofaster, gogoslick, gogotypes | Version4 |
| Editions 2023 / 2024 | gogo reflection backend, historical Open Struct API | Version5 |
| Edition 2026 | Same backend, custom JSON enum spellings in jsonpb | Version6 |

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

- Optional requires the gogo runtime. `nullable=false` and unsupported custom,
  cast, embedded, standard-type, face and union combinations are rejected.
- Editions currently use reflection output. Fast generators, gogo custom
  extensions and more than 64 required fields per message are rejected.
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
