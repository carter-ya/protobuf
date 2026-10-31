# Isolated standard-format qualification

This Go 1.23 module compares the explicit `jsonpb/standard` and
`textpb/standard` entrypoints with official Go Protobuf v1.36.12. It does not
add dependencies or raise the Go 1.15 requirement of the root module.

`FuzzText` and `FuzzJSON` compare acceptance, decoded message semantics through
an independent binary representation, and cross-parse SDK output. The fixture
covers signed/unsigned numbers, floating point, strings/bytes, booleans,
recursive messages, repeated scalars and maps. Inputs are limited to 4 KiB and
64 opening collection/message delimiters to keep resource-policy differences
outside the comparison. This is a focused fixture, not an exhaustive protocol
compatibility claim. Saved minimized failures run as ordinary regression tests.

The published [text grammar](https://protobuf.dev/reference/protobuf/textformat-spec/)
and official implementations have documented differences. The oracle explicitly
accounts for observed Go v1.36.12 differences without changing SDK policy:

- Vertical tab/form feed between tokens are legal whitespace. The oracle
  replaces them with spaces outside strings/comments before official parsing.
- Octal integer literals cannot initialize float fields. Official Go accepts
  some such spellings as decimal values; a string/comment-aware check rejects
  those spellings for the fixture's `f` field in the oracle.
- Official Go accepts `0F`/`1f` as booleans; these are float tokens and the
  published grammar permits only identifiers or unsigned integers for bools.
  The same check rejects them for the fixture's `flag` field.

Run deterministic regressions, both runtime implementations, and race checks:

```sh
cd compat/qualify
go test ./...
go test -tags purego ./...
go test -race ./...
```

Run bounded campaigns (CI uses two workers and 20 seconds per format):

```sh
GOMAXPROCS=2 go test -run '^$' -fuzz '^FuzzText$' -fuzztime=20s -parallel=2
GOMAXPROCS=2 go test -run '^$' -fuzz '^FuzzJSON$' -fuzztime=20s -parallel=2
```

The historical root text/JSON entrypoints retain their existing behavior and
are tested by the compatibility and runtime jobs separately.
