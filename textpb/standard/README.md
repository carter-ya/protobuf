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
The staged implementation removes all 223 originally recorded standard text
failures. All 909 pinned text cases now pass without skips; this qualifies that
suite, not every possible text input or gogo representation.
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

Declared reserved field names are ignored using the generated message descriptor,
including nested message descriptor paths. Skipped values retain lexical and
structural validation, and nesting is bounded. Other unknown names are errors.
Malformed descriptor data returns an error; messages without descriptors have no
reserved-name metadata. Schema facts are cached per Go message type.

Expanded Any URLs accept the reference tokenizer character set and percent
encodings, with validated prefix and type name. Whitespace and comments between
URL fragments are ignored. Standard expanded output is unquoted; invalid or
unresolvable expansion falls back to ordinary Any fields. Expanded URLs in other
message types are rejected. Historical URL parsing and printing are unchanged.

Group and group-like DELIMITED field names use descriptor spelling and scope.
Both the declared field name and valid message-name alias are accepted; aliases
share singular-field duplicate checks. Ordinary fields take precedence over
aliases. Standard printing uses the corresponding schema spelling, while legacy
Go tag spelling and extension-name behavior remain unchanged.

Byte escape widths follow the text specification: one to three octal digits,
and one to two hex digits after lowercase `\x`. Extra digits and following
characters remain literal. Out-of-range byte values and uppercase `\X` are
rejected. Historical entrypoints retain their full-width and uppercase behavior.

`Marshaler.RecursionLimit` and `Unmarshaler.RecursionLimit` bound text message
nesting; zero defaults to 10,000, matching the official Go text decoder. The root,
map entries, mapped wrappers and expanded Any messages count toward the budget.
Negative limits are invalid. Reserved/unknown values also observe the budget and
retain their existing additional bounds. Cyclic Go message graphs are rejected;
shared acyclic submessages are allowed. Encoder errors commit no partial output.
Non-struct messages require their own text hook. Custom hooks own their policy;
expanded Any payload bytes are decoded by the registered binary codec. Historical
text APIs keep their previous behavior and configuration layout.
