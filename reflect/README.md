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
