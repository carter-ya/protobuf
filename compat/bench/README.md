# Codec performance qualification

`run.py` measures encode, decode, allocated bytes/objects and generated Go bytes
for all five production generators. The same scalar/bytes, repeated primitive,
two child-message and eight-entry map fixture is used throughout. Payload sizes
are 64 and 4096 bytes; each record also carries the fixed non-payload fields.
The extension profiles add a registered native message extension with another
payload of the same size. Every workload verifies decoded fixture values before
timing. This baseline does not measure oneofs, JSON/text or custom representations;
the functional matrices qualify those separately.

Profiles compare original v1.3.2 against current proto2, and the fixed pre-Version13
revision against current Edition 2023 message extensions. Additional profiles
record proto3 optional and Editions 2023/2026. Compare revisions within the same
profile family and backend: layouts and semantics can differ across syntax
families. In particular, extension validation intentionally adds eager decode work
where the historical runtime deferred errors until `GetExtension`.

Use Python 3.12+, Go 1.27 and pinned protoc 36.2, from a clean checkout with the
v1.3.2 tag and pre-Version13 revision available:

```sh
python3 compat/bench/run.py /tmp/gogo-benchmark --count 3 --benchtime 100ms
```

The output directory must be new. `--profile` and `--backend` may be repeated for a
subset. Raw logs, exact revisions, environment, harness hashes, JSON and CSV
medians are retained. Original sources are extracted with `git archive`; only
snapshot developer dependencies are removed from their temporary `go.mod`.
Production Go sources are unchanged. The root module gains no dependencies.
Snapshots run with the same current toolchain, rather than their historical Go
compiler. CPU identity is taken from the benchmark process when available.

A shared fixture without a oneof also works with the original gogotypes
comparison plugin, which dereferences nil oneof indices when ordinary fields and
oneofs coexist. The original generator is neither patched nor substituted.

CI runs every current backend across proto2, optional, Edition 2026 and message
extensions once. This validates the harness and result inventory; one iteration
is not a meaningful speed measurement. It records artifacts and imposes no
hardware-dependent performance threshold. For a release decision, rerun the full
matrix on otherwise idle release hardware and retain the result directory.

`--current-revision COMMIT` runs the current-profile schemas against an archived
production revision while keeping the harness from the working checkout. This
supports matched comparisons across runtime fixes. The environment separates the
harness commit from the actual production source revision; snapshot production
source is unchanged and receives the same temporary dependency normalization.
