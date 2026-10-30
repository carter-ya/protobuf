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
| `customtype` | nil pointer | non-nil custom value whose wire payload may be empty | Pending validation |
| `stdtime`, `stdduration` | nil pointer | non-nil zero Go value; wire meaning follows the WKT mapping | Pending validation |
| `wktpointer` | nil pointer (nil slice for bytes wrapper) | non-nil zero / empty value | Pending validation |
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
