# Qualified compatibility and migration

Status date: 2026-10-31. This maintenance line starts from original gogo/protobuf
v1.3.2. The module remains `github.com/gogo/protobuf`; the root runtime requires
Go 1.15. Existing valid proto2/proto3 bindings and their public Go layouts remain
usable. Malformed-wire fixes enforce field-number and group bounds. Historical
JSON/text defaults are preserved through their original entrypoints.

The exhaustive feature inventory and remaining differences are tracked in
[the feature audit](feature-audit/README.md). Passing the existing conformance
profile does not imply that every Go-specific generation feature is implemented.

## Version and feature matrix

| Area | Qualified scope | Evidence |
| --- | --- | --- |
| Legacy consumers | Frozen v1.3.2 output and regenerated original schemas; original gogo plugin suite | `check.sh`; frozen etcd raft v3.5.0 and CockroachDB errors v1.8.1 consumers |
| Stable compiler | protoc 3.21.8 (release asset 21.8) and 36.2 | Compatibility CI on Go 1.23/1.27; Go 1.15.15 with 36.2 |
| Production backends | gogo, gogofast, gogofaster, gogoslick, gogotypes | Optional, Editions, auxiliary and binary differential matrices |
| Proto3 optional | Scalar/enum/message/bytes presence, synthetic oneofs, supported cast/custom/std rows | [Representation contract](optional/README.md) and five-backend default/purego tests |
| Editions | 2023, 2024 and 2026; gogo Open Struct API | Scalar/complex/auxiliary matrices, raw descriptors and independent oracles |
| Core Editions semantics | Presence, LEGACY_REQUIRED, UTF8 VERIFY/NONE, packed/expanded, open/closed enums, maps, oneofs, DELIMITED, extensions and option retention | `check-editions.sh`, `check-editions-fast.sh`, required-boundary and binary checks |
| Edition 2026 JSON | Custom enum names, aliases and empty strings | Pinned 36.2 descriptors, independent Go development oracle and Go 1.37 preview oracle |
| Native message extensions | Original occurrence boundaries, singular merges, required completeness after outer merges, decoded caches, map and byte storage | Runtime/direct-codec regressions; all five binary backends, both storage forms |
| Standard JSON/text | Explicit `jsonpb/standard` and `textpb/standard` entrypoints | Complete pinned 36.2 standard conformance profile and differential fuzz |
| Modern Go interoperability | Explicit detached Snapshot/Apply experiment, local Any/extensions and frozen downstreams | [C02 scope and restrictions](modern/README.md); no production ProtoReflect |
| Preview | protoc 37.0-rc1 and official Go v1.37.0-rc.1 | [Separate preview qualification](preview/README.md), isolated Go 1.26+ consumers |

Go 1.15 refers to this project's runtime and generated consumers. The stable
JSON/text differential module uses Go 1.23+ and official Go v1.36.12. The existing
2026/C02 oracle pins development commit `0c373ff94f29`; the newer official Go
preview requires Go 1.26. None of those official runtimes becomes a dependency
of the production gogo module. Historical compiler/toolchain combinations absent
from the current matrix are not qualified by the retained old workflow file.

The stable protocol reference is [Protobuf 36.2](https://github.com/protocolbuffers/protobuf/releases/tag/v36.2).
[37.0-rc1](https://github.com/protocolbuffers/protobuf/releases/tag/v37.0-rc1)
and [Go v1.37.0-rc.1](https://github.com/protocolbuffers/protobuf-go/releases/tag/v1.37.0-rc.1)
remain previews. Their 20 bundled schemas differ only in two reviewed JSON
option Go-package additions, excluding source comments. Preview compiler/runtime
qualification is separate from the stable C++ conformance run.

## Standard behavior is explicit

| Pinned 36.2 profile | Binary/JSON failures | Variable binary/JSON outcomes | Binary/JSON skips | Text failures/skips |
| --- | ---: | ---: | ---: | --- |
| Standard | 0 of 5,649 cases | 0 | 4 unstable-Edition cases | 0 / 0 of 909 cases |
| Historical defaults | 128 recorded failures | 16 oneof/null cases | The same 4 cases | 223 / 0 of 909 cases |

The historical inventory is retained so updating the runtime does not silently
select different JSON/text policies. The standard profile uses separately
generated binary code with `merge_oneof_messages=true,merge_required_fields=true`
and explicit standard
JSON/text entrypoints. It passes 5,645 binary/JSON cases and all 909 text cases.
The four skipped cases use `EditionUnstable`, outside the supported 2023–2026
range. No stable-Edition case is skipped. Exact inventories live in
[the gate](conformance/README.md), [standard records](conformance/failures-standard.json)
and [legacy records](conformance/failures.json).

For new standard behavior, use imports such as:

```go
import (
    jsonstandard "github.com/gogo/protobuf/jsonpb/standard"
    "github.com/gogo/protobuf/proto"
    textstandard "github.com/gogo/protobuf/textpb/standard"
)

func parseJSON(input string, message proto.Message) error {
    return jsonstandard.UnmarshalString(input, message)
}

func printText(message proto.Message) (string, error) {
    return (&textstandard.Marshaler{}).MarshalToString(message)
}
```

Choose each entrypoint deliberately; ordinary `jsonpb` and `proto` text APIs
retain their historical configuration layouts. Custom JSON/text hooks own their
representations and are not covered by a blanket standard-conformance claim.
Standard JSON has a 100-container bound; standard text has a configurable
recursion limit with default 10,000. Binary differential inputs are bounded at
4 KiB; those campaigns do not qualify every resource boundary or possible schema.

## Migrating an existing project

First consume this fork's runtime while retaining existing imports and frozen
bindings. For a local checkout, use a consumer replacement:

```sh
go mod edit -replace=github.com/gogo/protobuf=/absolute/path/to/protobuf
```

Build generators from the same selected checkout as the runtime. An unqualified
`go install github.com/gogo/protobuf/...@latest` selects the original module's
published versions rather than this checkout.

```sh
cd /absolute/path/to/protobuf
mkdir -p /absolute/path/to/tools
go build -o /absolute/path/to/tools/protoc-gen-gogo ./protoc-gen-gogo
```

Then regenerate only when adopting optional/Editions or another new generated
policy. Use pinned protoc and preserve the application's existing output/import
mapping flags. Proto2/proto3 message-oneof merging requires the explicit
[generator parameter](oneof-merge/README.md); Editions selects that policy
through its metadata. Regenerate enclosing and referenced codecs together:
precompiled imported bindings retain their own decode policy and methods.

New generated files assert internal runtime capabilities. These numbers are not
module release versions; keep the generated guard and use the matching runtime.

| Guard | Capability |
| --- | --- |
| Version3 | Historical v1.3.2 bindings |
| Version4 | Proto3 optional presence |
| Version5 / 6 | Editions semantics / 2026 JSON enum names |
| Version7 | Required fields beyond the first 64 |
| Version8 / 9 | Generated closed-enum filtering / group and declaration-scoped extension helpers |
| Version10 / 11 | Optional casts / custom and standard mappings |
| Version12 | Explicit oneof message merge metadata |
| Version13 | Native Editions extension validation, cached merges and deferred enclosing-graph completeness |
| Version14 | Required initialization after final logical-message merging |

Version13 also applies to a newly generated enclosing Editions file that reaches
extensible children but declares no ranges itself. Its fast decoder uses the
shared table path for that graph; other fast field paths remain generated.
Version14 applies to newly generated Editions messages that reach required fields,
including imported dependencies and native extension payloads. Their fast decoders
use the shared table path to defer completeness until the final message. For
proto2/proto3 this is the explicit `merge_required_fields=true` parameter; unmarked
bindings keep their historical required policy. Regenerate the complete dependency
graph and retain independent presence for every required field. Non-nullable
required scalars, flattened singular message fields and mapped message paths
with required descendants are rejected.
See [required merge qualification](required-merge/README.md).

An older runtime must fail to compile new feature code. Frozen legacy bindings
continue to compile without regeneration.

## Boundaries and release decisions

Optional representations are limited to the supported rows in the linked
contract. New optional presence cannot use `nullable=false`, incompatible custom
or message casts, embed/face/union, or gofast's alternate runtime. Editions keeps
its separate gogo option allowlist; custom/cast/nullable and unsafe codec options
outside that list fail generation rather than emitting partially supported code.
Legacy schemas retain their established gogo mappings and helpers.

Fresh native float/double Compare methods and the maintained scalar wrappers
use [NaNs-before-numbers ordering](compare-floats/README.md), with equal NaNs
continuing to later elements/fields. The historical Go floating-point rule for
[proto.Equal](../proto/equal.go) and generated equality is retained. External
frozen Compare implementations need regeneration for the ordering fix.

Explicit Go HYBRID/OPAQUE API requests, enum-prefix stripping and unsupported Go
features produce diagnostics. This backend preserves its Open Struct API. The
C02 conversion experiment rejects MessageSet, unknown closed-enum reflection
mismatches, lossy mappings and incomplete descriptor metadata; it is not a
released transparent V2 adapter. Unknown or future Editions are rejected.

The [performance baseline](bench/BASELINE.md) retains all five backends, actual
wire/code sizes and raw measurements. The [map metadata follow-up](bench/MAP-CACHE.md)
removes the measured reflection-map allocation regression. Registered native
extension validation still adds decode work; no hardware-independent speed
threshold is claimed. These measurements qualify their documented fixtures.

A release candidate must pass every current CI check and a clean review of the
exact submitted head. Keep compiler/runtime pins, frozen snapshots, schema audits,
conformance inventories, fuzz corpus and downstream checks reproducible. Publishing
a versioned release or tag is a separate maintainer action. Subsequent upstream
stable releases require a new schema/default audit and independent conformance
qualification before changing the support matrix.
