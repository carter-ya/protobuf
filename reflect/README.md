# Optional live Go V2 reflection

This module exposes native gogo message storage through the public V2
`protoreflect` interfaces. It requires Go 1.23; the parent runtime remains Go
1.15 and does not import this module. Its official runtime is pinned to the
same qualified development revision as the Edition 2026 integration experiment.
Stable and preview qualification remain separate from the parent module.

```go
modern, err := gogoreflect.Wrap(existingGogoMessage)
if err != nil { return err }
wire, err := proto.Marshal(modern) // google.golang.org/protobuf/proto
```

`Wrap` validates storage before returning a live view. Reflection changes modify
the original message, and direct Go changes are visible through populated
views. `Unwrap` recovers the native gogo value. Message allocation, presence,
unknown fields, pointer children, scalar/message lists, native maps and oneofs
use their actual Go storage. Empty `Get` composites are read-only; `Mutable`
and `NewField` provide mutable values. The API is not Snapshot/Apply conversion.

The qualified native profile includes proto2/proto3, optional synthetic oneofs,
closed enums, groups, required fields checked after final merging, and registered
map-backed extensions. Regenerate legacy closed-enum bindings with
`enforce_closed_enums=true`; unqualified legacy tags are diagnosed. Unknown
closed enum scalar/list/map/oneof/extension values retain native wire semantics.
The decoder processes occurrences in wire order: closed-enum scalar/map fields
use the qualified native decoder while other fields use V2's reflective decoder.
This preserves resolver, recursion, oneof and merge behavior without inheriting
V2's known closed-enum decoding mismatch. DiscardUnknown preserves existing
unknowns during Merge. Required checking remains the final V2 initialization
check, and AllowPartial supports partially initialized messages.

`NewRegistry(roots...)` provides an explicit V2 message/extension resolver, also
including reachable native message types. Pass Any payload types as roots, and
use the registry with protojson/prototext or Any unmarshal options. No V2 global
registrations are installed. Binary decoding with the default global resolver
also recognizes native gogo extension bindings; an explicitly supplied resolver
is authoritative. The resolver's extension descriptors must match the canonical
native declarations. Serial late native extension registration remains visible.

Custom/std mappings, flattened message values, byte-backed extensions,
MessageSet, disabled unknown storage, mismatched descriptors and lost explicit
presence return errors. These representation boundaries are explicit and
transactional at Wrap/registry qualification; do not substitute detached dynamic
snapshots and count them as live reflection. This module does not add methods to
frozen generated files. Unregistered native extension wire is exposed as unknown
fields in stable field-number order, preserving occurrence order within each
extension number; legacy map storage does not retain cross-number wire order.

Finish gogo descriptor/type registration before using reflection. Descriptors
are canonical within this module; known historical gogo/descriptor import aliases
share one identity. Conflicting same-path metadata is rejected.
Concurrent reads of an initialized message are supported; synchronize writes and
concurrent registration as required by the native gogo runtime. Byte slices and
native message values retain protobuf aliasing semantics; reflection is not a
thread-safety layer or an immutable copy.

`sh compat/check-live-reflect.sh` verifies stable fixture regeneration and runs
default, purego and race checks, including the official `prototest.Message`
contract suite and independent binary/JSON/text, Clone/Merge/Equal tests. The
root's historical compatibility and real-project matrix remain separate gates.

## Generating native V2 methods and HYBRID

Use `--gogo_out=proto_reflect=true:OUT` to add native `ProtoReflect` while
retaining the OPEN field API. Use `default_api_level=API_HYBRID` to also generate
setters, presence/clear methods, oneof case queries and builders. The same flags
work with all five maintained generators. Editions can select
`features.(pb.go).api_level = API_HYBRID` at file or message scope; explicit OPEN
and nested overrides are honored. `default_api_level=API_OPAQUE` or source
`API_OPAQUE` emits private protobuf fields with accessors/builders. A HYBRID file
also emits a `_protoopaque.pb.go` alternate selected with `-tags=protoopaque`.
`apilevelMfile.proto=API_OPEN|API_HYBRID|API_OPAQUE` sets a file default; explicit
source features take precedence. Historical default OPEN generation remains
unchanged, including the default API for Editions.

Selected generated types implement both gogo and V2 Message directly. V2
`Interface`, `New`, `Clone` and `Wrap` preserve the actual generated Go type.
HYBRID retains exported fields and legacy getters; builders retain protobuf
pointer/slice/map aliasing and setters preserve official nil/presence semantics.
New API name collisions are diagnosed rather than renaming existing Go fields.

The default generator output and the root module's Go requirement are unchanged.
Opting in adds this module and V2 dependencies, the version-18 optional-runtime contract, closed
native enum semantics and final required-field validation. Convenience backends
such as gogofaster normally choose lossy value storage and disable unknowns;
the selected profile instead defaults to native pointer/unknown storage. Review
these opt-in field-type changes when migrating. Explicit incompatible schema
options fail generation; imports are validated across the reachable graph at
runtime qualification. Registration must finish before reflection is used.

Option-only dependency descriptors travel with the generated file as compressed
metadata. `RegisterFileSet` validates duplicate registrations transactionally;
it does not register V2 global types. Use `NewRegistry` explicitly for Any and
Edition 2026 custom enum JSON metadata, including nested imported options.

The qualified HYBRID form retains exported fields. Its opaque alternate and
explicit OPAQUE use the same live native identity/storage contract, including
legacy JSONPB, SetDefaults, required validation and expanded Any text. OPAQUE
numeric-scalar builder inputs are copied; string-pointer and message/slice/map
ownership follows the official API. Storage retains native pointers and eager
decoding; no reduced-memory or lazy-decoding performance claim is made.

These capabilities require root version 18. The qualified published combination
is stable root `v1.4.0` at `057d06490a721bea0fceb39c5c088bc1ef3085e8` and this optional
module `reflect/v0.1.0-rc.2` at `3af6d45fd9dbe0ff1249afa246a44caf995b29a1`.
The optional module remains RC; RC1 excludes OPAQUE. Match the root generator
and runtime version, and select the separately qualified optional version.
The nested module's local replace directive is not inherited by applications.

For a source checkout beside your application:

```go
require github.com/gogo/protobuf/reflect v0.0.0
replace github.com/gogo/protobuf => ../protobuf
replace github.com/gogo/protobuf/reflect => ../protobuf/reflect
```

For remote builds, use the exact root and optional fork versions in the
[installation guide](../release/INSTALL.md). Separate source revisions are
qualified explicitly; arbitrary mixed versions are not covered. The version-18
compile guard detects an older root runtime. Custom generator wrappers that
normalize descriptors should use `command.ReadWithSources` and
`command.GenerateWithSources` so shared option metadata remains source-stable.


## Release pairing

The existing module [reflect/v0.1.0-rc.2](https://github.com/carter-ya/protobuf/releases/tag/reflect/v0.1.0-rc.2)
retains its original source, dependency pin and minimum root `v1.4.0-rc.2`.
Stable root [v1.4.0](https://github.com/carter-ya/protobuf/releases/tag/v1.4.0)
satisfies that minimum and passed the complete cross-revision qualification.
Earlier RC2 root/reflect pairing remains immutable. Later development commits
are qualified independently and do not move either tag.

This RC retains the exact development runtime revision in go.mod for Edition
2026 option/JSON semantics. The parent runtime has no V2 dependency. See the
[dependency decision and promotion gates](../release/README.md#official-runtime-decision)
before upgrading or attempting a downgrade to the stable reference.

### Source fixes after RC2

The current source avoids repeated unknown-buffer copies during closed-enum
merge/discard decoding and ordinary unknown-field decoding, including messages
with extension ranges. Closed-enum occurrences use an empty carrier so the
accumulated lists/maps/extensions are not revalidated or re-encoded for each
field. Final required-field completeness is checked by the outer V2 operation.
These fixes are not in the immutable published reflect RC2 module.
RC2 may allocate disproportionately for inputs containing many unknown or known closed-enum occurrences,
including Merge+DiscardUnknown with a large existing unknown buffer. Bound input
sizes and occurrence counts when using that RC with untrusted wire data, or use
a separately qualified full source revision. Root v1.4.1 publication does not
republish or silently change the optional module. A future optional-module
release needs its own versioned installation qualification.
