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
and nested overrides are honored. OPAQUE is rejected.

Selected generated types implement both gogo and V2 Message directly. V2
`Interface`, `New`, `Clone` and `Wrap` preserve the actual generated Go type.
HYBRID retains exported fields and legacy getters; builders retain protobuf
pointer/slice/map aliasing and setters preserve official nil/presence semantics.
New API name collisions are diagnosed rather than renaming existing Go fields.

The default generator output and the root module's Go requirement are unchanged.
Opting in adds this module and V2 dependencies, a version-17 runtime guard, closed
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

The qualified HYBRID form retains exported fields. The official `protoopaque`
alternate build and OPAQUE layout are deferred together; this implementation
does not provide an opaque alternate file for that build tag.

When consuming this maintained fork, replace both modules at the same qualified
revision. The nested module's local `replace` directive is not inherited by
applications. For a checkout beside your application:

```go
require github.com/gogo/protobuf/reflect v0.0.0
replace github.com/gogo/protobuf => ../protobuf
replace github.com/gogo/protobuf/reflect => ../protobuf/reflect
```

For remote builds, pin the corresponding root and `/reflect` replacements to
the same reviewed commit in `github.com/carter-ya/protobuf`; do not invent a
nested-module release tag. Generate with the same revision. The version-17
compile guard detects an older root runtime. Custom generator wrappers that
normalize descriptors should use `command.ReadWithSources` and
`command.GenerateWithSources` so shared option metadata remains source-stable.


## Release pairing

The proposed initial module release is `reflect/v0.1.0-rc.1`, paired with root
`v1.4.0-rc.1` at the exact same commit. This preparation does not publish either tag.
The minimum root requirement is explicit in go.mod; an application must replace
both canonical module paths with their corresponding remote fork versions.
The local replace is for repository development and is ignored by dependent
applications. Follow [the remote installation guide](../release/INSTALL.md).

This RC retains the exact development runtime revision in go.mod for Edition
2026 option/JSON semantics. The parent runtime has no V2 dependency. See the
[dependency decision and promotion gates](../release/README.md#official-runtime-decision)
before upgrading or attempting a downgrade to the stable reference.
