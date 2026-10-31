# Optional representation contract

This matrix concerns **proto3 optional** on the gogo runtime. Editions have a
separate allowlist; permitting an optional mapping does not enable it for
Editions or gofast's alternate runtime. Existing proto2/proto3 schemas and
frozen generated files keep their historical contract.

| Representation | Absent | Present zero / empty | Eligibility |
| --- | --- | --- | --- |
| Default scalar or enum | nil pointer | non-nil pointer to zero | Supported |
| Default bytes | nil slice | non-nil empty slice | Supported |
| Default message | nil pointer | non-nil empty message | Supported |
| Scalar/enum/bytes `casttype` | nil pointer, or nil slice for bytes | typed zero pointer / non-nil empty bytes | Supported; same underlying Go kind and width as the schema field |
| Message `casttype` | unvalidated message API | unvalidated codec methods | Rejected |
| Bytes `customtype` | nil pointer | non-nil custom value whose wire payload may be empty | Supported; requires a compatible custom codec |
| Message/scalar `customtype` | unvalidated mapping | unvalidated mapping | Rejected |
| `stdtime`, `stdduration` | nil pointer | non-nil zero Go value; wire meaning follows the WKT mapping | Supported on Timestamp / Duration respectively |
| `wktpointer` | nil pointer | non-nil zero / empty value (`*[]byte` for BytesValue) | Supported on the nine scalar wrapper messages |
| `nullable=false` | no independent representation | cannot distinguish absence from zero | Rejected |
| `embed`, `face`, `onlyone` | unvalidated generated API / method interactions | no maintained contract | Rejected |

New mappings must pass separately through gogo, gogofast, gogofaster, gogoslick
and gogotypes. A compatible user type must still satisfy the established
mapping's Go type and codec requirements. Supporting a mapping never means
accepting arbitrary incompatible user implementations.

Each enabled combination must verify:

- Empty input stays absent; explicit zero/empty emits a field and survives
  binary, legacy JSON, standard JSON and text roundtrips.
- Size agrees with Marshal; fast direct codecs agree with the reflection
  runtime and an independently decoded wire message.
- Merge from absent preserves the destination; Merge from explicit zero/empty
  replaces its old value. Clone does not alias mutable values. Equality and
  generated comparison distinguish absence from explicit presence.
- Raw descriptors retain proto3_optional, synthetic oneofs and mapping options.
- Invalid UTF-8 strings are rejected; malformed or incompatible combinations
  fail with a file/message/field diagnostic.
- Historical generated snapshots and downstream consumers still pass unchanged.

`generator/presence_test.go` freezes the current eligibility and descriptor
validation rules. Integration tests exercise only combinations listed as
supported. Further rows are opened only together with their implementation and
consumer tests, not by removing the representation guard wholesale.

`sh compat/check-optional-mappings.sh` generates isolated mapping consumers for
all five public backends under both CI protoc versions. Scalar casts require
the matching underlying Go kind and width (including int32 for enums); arbitrary
narrowing casts are not covered by this guarantee. Named enum integers without a
String method use legal numeric JSON. New mapping files require runtime Version10;
ordinary optional files keep Version4. Editions cast mappings remain rejected.

Optional scalar getters now check field presence before dereferencing the
pointer, including the previously supported default representation. The maintained
optional snapshots are regenerated for that fix; frozen v1.3.2 snapshots are unchanged.

The optional mapping gate runs each generated consumer in both the default
and `purego` configurations. Pure-Go table merging preserves the actual named
field type rather than asserting a builtin scalar pointer. The per-field
Clone/Merge checks cover all scalar families, enum casts, string casts and
nil/empty/nonempty bytes, including copy ownership and explicit zero updates.
This runtime path is shared with the App Engine and JS pointer implementations.
Codec tables likewise adapt named fields to temporary builtin slots in pure-Go
builds and restore their named types after decoding. Legacy fields without
optional cast metadata keep their existing paths.

## Custom and standard mappings

Custom bytes and standard mappings require runtime Version11. The matrix tests
both reflection and generated codecs, all nine wrappers, split/duplicate WKT
wire bodies, binary/text oracle roundtrips, private mutable custom state,
nullable bytes wrappers and Go 1.15-compatible auxiliary methods. Explicit
unsafe codecs and conflicting representation options are rejected.

A bytes custom type must provide the established Marshal/MarshalTo/Size and
Unmarshal interfaces. Enable Equal/Compare/Populate/GoString only when the type
also supplies those methods/factories. Runtime equality compares the encoded
opaque byte payload. Clone/Merge copy it through the codec into a fresh value;
private state is never traversed. Custom Equal/Compare methods must agree with
that payload contract, and custom JSON hooks must implement valid bytes JSON.
Invalid custom implementations are not repaired: Clone/Merge cannot return an
error and panic if a custom codec fails to roundtrip its own valid value.

Mapped standard fields expose Go values: present values replace prior mapped
values in proto.Merge, including zero and empty bytes. Binary decoding instead
merges known inner WKT fields across duplicate wire occurrences, including
UnmarshalMerge into a populated destination. Empty Timestamp wire input means
the Unix epoch; Go's zero time represents year 1. Nil and empty inner BytesValue
slices are equivalent, while the outer pointer determines message presence.
The new fast WKT helpers share the runtime codec for UTF-8 and floating-point
bit correctness, including negative zero; no speedup is promised for this path.

These explicit mappings cannot retain unknown fields *inside* a WKT or expose
its inner field presence. Timestamp/Duration must fit their mapped Go value
and validity constraints; time.Duration cannot represent the full Protobuf
Duration range. Keep ordinary message fields when that metadata or range is
needed. The default message representation retains its separate contract;
Editions representation mappings and gofast remain unsupported.

Optional custom/Timestamp/BytesValue equality checks presence in both directions
before invoking value methods or dereferencing. Equal and VerboseEqual have the
same presence contract. Optional wrapper text codecs retain the declared message
shape (`field { value: ... }`), including an empty body and a colon-free string
or bytes wrapper. Compact and expanded text interoperate in both directions with
protoc; flattened scalar input is rejected for these mapped message fields.
Historical non-optional mapping text and generated equality remain unchanged.
