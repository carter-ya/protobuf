# Explicit definition-scoped closed enums for legacy schemas

`enforce_closed_enums=true` selects the Protobuf enum definition's open/closed
policy when regenerating proto2/proto3 bindings. Unmarked generation and frozen
v1.3.2 bindings retain historical enum decoding. Editions already selects its
resolved enum policy automatically.

```sh
protoc --gogo_out=enforce_closed_enums=true,merge_oneof_messages=true,merge_required_fields=true:out schema.proto
```

Preserve the project's existing backend, import mappings and output flags. All
five gogo-runtime backends support the option. Regenerate the enclosing and
referenced codecs together: precompiled imported bindings keep their own decode
policy. For the complete standard profile, also select the independent oneof
and required merge options and explicit standard JSON/text entrypoints.

Closed unknowns do not set singular/required/oneof fields or overwrite a prior
known value. Lists retain known elements; packed unknowns are preserved as
unpacked unknown varints. A closed-enum map entry with an unknown value is kept
as an unknown field. Registered scalar/list enum extensions follow the same
policy, including registrations after a table was initialized. Nonzero first
proto2 enum values remain the default for unset fields. Proto2 UTF-8 policy is
unchanged. Imported proto3 enums remain open because openness belongs to the
definition, not the referencing file.

The option enables unknown-field storage and enum presence before convenience
backends install their historical defaults. Synthetic map values keep implicit presence; filtering retains the complete
entry when its enum is unknown. Explicit source options are never
overridden: disabling unknown storage or flattening singular closed enums fails
with a diagnostic. Custom enum representations and alternate runtimes are not
qualified. Native enum names and old generated layouts outside this explicit
policy retain their existing behavior. Opted-in code asserts runtime Version15.
Fast decoding uses the shared table codec to apply the same filtering policy as
reflection; historical fast output is unchanged.

`sh compat/check-legacy-enums.sh` tests five backends, standard/legacy controls,
default/purego builds, map/byte extension stores, required presence, nested
messages, imported open/closed enums, direct generated decoding, merge-into-prior
state, standard JSON/text and a compile-negative old-runtime guard. Protoc
3.21.8 omits only the unsupported Editions import; 36.2 and 37.0-rc1 exercise it.
Each storage/compiler matrix has 20 consumer runs and 20 unsupported-option
checks. Eleven wire cases per standard backend/build are cross-read by the
independent C++ compiler's decoder before and after Go re-encoding.

The imported proto3 enum case uses the published definition-scope rule as its
expectation. It is intentionally excluded from the C++ cross-read comparison:
[C++ has a documented proto2-imports-proto3 enum exception](https://protobuf.dev/programming-guides/enum/).
The official Go dynamic runtime's own closed-enum exception likewise prevents
using it as the sole closed-enum oracle. Cross-read equality supplements explicit
presence/unknown assertions; it is not a blanket all-schema conformance claim.
