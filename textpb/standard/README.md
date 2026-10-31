# Explicit standard text entrypoint

This package introduces standard text rules without changing the historical
`proto.UnmarshalText`, `proto.MarshalTextString` or TextMarshaler configuration
layout. It is the text counterpart of `jsonpb/standard`. The initial coverage
is deliberately recorded by the upstream conformance gate, not a claim of full
text-format conformance.

```go
var message pb.Message
err := standard.UnmarshalString(input, &message)
text, err := (&standard.Marshaler{Compact: false}).MarshalToString(&message)
```

The initial implementation covers decimal floating-point literals with `f/F`,
case-insensitive special values, overflow to infinity, signed underflow and
negative-zero output. Hexadecimal/octal-looking float literals are rejected.
Numeric values of closed proto2/Edition enums must be declared. UTF-8 checking
uses the field's resolved policy; UTF8_NONE retains byte values. Nil root
messages return errors. Output is buffered until encoding succeeds.

Use keyed configuration literals. Compact and ExpandAny mirror `proto.TextMarshaler`. Custom
encoding.TextMarshaler/TextUnmarshaler hooks own their representations and must
satisfy the selected policy. Unmarshal resets the destination, matching the
legacy parser's ownership contract. Raw descriptors and generated APIs are not
changed by selecting this package.

The pinned Protobuf 36.2 suite runs legacy and standard text independently.
The first implementation removes 125 standard-only failures; 66 standard text
failures remain, including reserved names, Any URL spelling and group casing.
Legacy behavior and its 223-failure baseline remain separate. See
`compat/conformance/README.md` for the full test inventory and exact failures.

`PrintUnknownFields` defaults to false. When enabled, unknown fields use valid
text syntax and fixed-width hexadecimal fixed values. Complete nonempty wire
payloads can be rendered as nested messages, following the reference printer's
heuristic; recursive interpretation is bounded at 100 levels. Malformed unknown
wire is an encoding error. The legacy debug-comment format is unchanged.

The numeric rules follow the official [text-format specification](https://protobuf.dev/reference/protobuf/textformat-spec/).

Unicode escapes must encode scalar values; surrogate halves and pairs are rejected
for both strings and bytes. Raw byte escapes retain their byte semantics.
