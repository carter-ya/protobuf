# Explicit standard JSON behavior

Use `github.com/gogo/protobuf/jsonpb/standard` when adopting the stricter
Protobuf JSON contract. The existing `jsonpb` entrypoint keeps its historical
behavior and configuration layout. The new Marshaler and Unmarshaler expose the
same configuration fields; existing message types and imports remain usable.

```
var value MyMessage
err := standard.UnmarshalString(input, &value)
output, err := (&standard.Marshaler{}).MarshalToString(&value)
```

This entrypoint consumes one complete document, resets the destination, validates
UTF-8 and surrogate pairs, and rejects duplicate keys, alternate spellings of
the same field and conflicting oneof members. Null ordinary fields remain unset;
null repeated/map elements are rejected except for Protobuf Value/NullValue.
Absent presence-bearing fields are omitted even with EmitDefaults. Nil binary
map message values serialize as empty messages. Numeric strings do not permit
leading/trailing whitespace.

Duplicate-field rejection is one of the permitted Protobuf JSON behaviors when
the parser does not preserve input order. The legacy parser's arbitrary oneof
iteration is not inherited by this entrypoint. JSONPB custom hooks remain the
message author's responsibility; they receive the historical configuration and
must implement any custom representation consistently with the standard.

These contracts do not claim full conformance. The pinned
upstream gate runs both entrypoints separately. A03 removed
52 additional stable failures and all 16 variable legacy oneof/null outcomes;
remaining binary, JSON and text limitations are individually recorded in
`compat/conformance/failures-standard.json`. A04 adds full-range Duration, validated Timestamp, FieldMask strings and
Any special-type handling. It removes 44 further stable failures, with
independent official C++ and Go interoperability checks. No modern official runtime dependency is added.

The completed standard 36.2 profile now has zero failures across 5,649
binary/JSON cases (four deliberately unsupported unstable-Edition skips) and
all 909 text cases pass without skips. This profile uses separately generated
message-oneof merge policy plus the explicit standard JSON/text entrypoints;
selecting this JSON package alone does not retrofit imported binary decoders.
Legacy mode retains its separate historical baseline and behavior. See
[the current support matrix](../../compat/SUPPORT.md) for the full contract.

With `AllowUnknownFields`, unknown enum **names** are discarded: an optional
field/oneof stays absent, repeated elements are omitted and map entries are
omitted. Known names and ordinary numeric enum handling retain their existing
rules. Malformed enum input is still an error. Legacy jsonpb retains its prior
unknown-enum behavior even when AllowUnknownFields is set.

Standard JSON permits at most 100 nested object/array containers, counting the
root container. Both input and emitted JSON are checked. Scalar leaves do not
consume another container level. Legacy jsonpb keeps its existing nesting
behavior. This is the standard entrypoint's documented resource bound.
