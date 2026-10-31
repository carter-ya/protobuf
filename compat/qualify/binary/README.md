# Binary differential qualification

`compat/check-binary-fuzz.sh` generates independent Go 1.23 test modules for
`gogo`, `gogofast`, `gogofaster`, `gogoslick` and `gogotypes`. The root module
continues to require Go 1.15. Each module uses the unmodified Protobuf 36.2
`TestAllTypesEdition2023` schema and this directory's separate `closed.proto`.
The latter adds closed enums, presence, groups, maps, oneofs and extensions.
Generation opts into `merge_oneof_messages=true`; imported frozen bindings do
not acquire that policy automatically.

The pinned C++ runner exposes a private framed `wire_oracle` mode. Both fixtures
use native C++ generated parsing. A dynamic C++ map representation can retain
duplicate entry records before synchronization and produced a false difference;
the saved duplicate-empty-entry corpus prevents that oracle regression.
The original official suites and their fixtures remain unchanged.

For inputs accepted by C++ and official Go v1.36.12, SDK decoding must succeed.
Official Go only resolves acceptance disagreements: C++ accepts some overflowing
tenth varint bytes that Go and the SDK reject. C++ is the sole semantic oracle,
including unknown closed-enum values; the Go dynamic decoder is not used for
that comparison. SDK output must preserve C++ message semantics and agree with
`proto.Size`. NaN values compare equal. Rejected C++ inputs still exercise SDK
parsing to detect panics, but differing parser resource limits are not treated
as an acceptance requirement. This does not assert that every malformed input
is rejected; targeted validation regressions remain separate.

The deterministic seeds and saved corpora cover scalar bounds, packed/unpacked
fields, unknown enums, maps, merging/replacing oneof members, extensions,
mismatched wire types, truncated groups and maximum field tags. Fuzz input is
limited to 4 KiB. CI runs deterministic default/purego/race tests and a bounded
two-worker 20-second campaign under both default and purego for every backend.
This qualifies the recorded fixtures and execution budget, not all possible
schemas, imported decoder policies or gogo custom mappings.

Build and run using absolute output/tool paths:

```sh
python3 compat/conformance/build-runner.py /tmp/protobuf-conformance
export PROTOBUF_CONFORMANCE_SOURCE=/tmp/protobuf-conformance/protobuf-36.2
export PROTOBUF_CONFORMANCE_RUNNER=/tmp/protobuf-conformance/bin/conformance_test_runner
export BINARY_FUZZ_OUTPUT=/tmp/protobuf-wire-results
sh compat/check-binary-fuzz.sh
```

`BINARY_FUZZ_TIME` overrides the per-backend/per-runtime campaign duration.
Result modules and logs are retained for reproduction, including minimized
failure corpora. Copy a confirmed new corpus into this directory's `testdata`
before submitting its fix.

Regenerated extensible Editions files and extension declarations require runtime
Version13 and carry `edition_ext` metadata. Registered native message extensions
are checked during outer decoding for wire, UTF-8 and required-field errors while
the encoded store remains lazy. Singular fragments merge before validation;
repeated messages are checked separately. Nested required-fragment recovery uses
the table decoder and validates the merged graph. Decoded cached values are
materialized before new fields are appended; an invalid prior cached value
returns an error without dropping it. Unknown registrations remain opaque, and
unmarked old descriptors retain their existing policy. Native checks exclude
custom/cast/standard representation overrides.

The `goproto_extensions_map[_all]=false` representation is also qualified for
Editions. The harness repeats deterministic/default/purego/race tests for all
five generators with byte storage and runs two-worker five-second campaigns
under both runtime implementations. `BINARY_BYTE_FUZZ_TIME` overrides that
additional duration. Byte stores preserve every message/repeated occurrence.
