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

A03 establishes these rules but does not claim full conformance. The pinned
upstream gate runs both entrypoints separately. Standard mode currently removes
52 additional stable failures and all 16 variable legacy oneof/null outcomes;
remaining binary, JSON and text limitations are individually recorded in
`compat/conformance/failures-standard.json`. WKT format/range work follows in
A04. No modern official runtime dependency is added.
