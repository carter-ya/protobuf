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

This initial layer supports native proto3 fields, including optional synthetic
oneofs and named scalar representations with matching primitive kinds.
Closed enums and extensions deliberately return errors until the decoder and
extension bridges are qualified. Custom/std mappings, flattened message values,
disabled unknown storage, mismatched descriptors and lost explicit presence
also return errors. Unsupported representations must be diagnosed rather than
silently losing data. This module does not add methods to frozen generated files
or install anything into V2's process-wide registries.

Finish gogo descriptor/type registration before using reflection. Descriptors
are canonical within this module and conflicting same-path metadata is rejected.
Concurrent reads of an initialized message are supported; synchronize writes and
concurrent registration as required by the native gogo runtime. Byte slices and
native message values retain protobuf aliasing semantics; reflection is not a
thread-safety layer or an immutable copy.

`sh compat/check-live-reflect.sh` verifies stable fixture regeneration and runs
default, purego and race checks, including the official `prototest.Message`
contract suite and independent binary/JSON/text, Clone/Merge/Equal tests. The
root's historical compatibility and real-project matrix remain separate gates.
