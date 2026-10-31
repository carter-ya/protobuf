# Separate preview qualification

The stable reference remains [Protobuf 36.2](https://github.com/protocolbuffers/protobuf/releases/tag/v36.2).
This job separately pins [protoc 37.0-rc1](https://github.com/protocolbuffers/protobuf/releases/tag/v37.0-rc1)
and [Go Protobuf v1.37.0-rc.1](https://github.com/protocolbuffers/protobuf-go/releases/tag/v1.37.0-rc.1).
It does not promote a release candidate to stable support. The upstream Go
preview requires Go 1.26; only temporary consumer modules adopt that minimum.
The root module continues to require Go 1.15 and its dependencies do not change.

With Python 3.12+, Go 1.26+ and protoc 36.2 on PATH, run:

```sh
python3 compat/preview/run.py /tmp/gogo-preview
```

The result directory must be new. The script downloads the official compiler
archive for Linux amd64/arm64 or macOS amd64/arm64 and checks its pinned SHA-256
before extracting it. `--archive` reuses a local official archive with the same
checksum check; `--stable-protoc` selects the stable compiler. `GOTOOLCHAIN=local`
prevents an implicit Go toolchain upgrade. The report retains tool versions,
archive and harness hashes, source revision, working-tree state, individual
check outcomes, schema descriptor sets and logs. CI retains reports and any
failing fuzz corpus.

The schema audit compares all 20 bundled official proto files without source
comments. It permits only two reviewed additions: `options.go_package` becomes
`google.golang.org/protobuf/types/jsonoptionspb` in `json_options.proto` and
`json_enumvalue_options.proto`. All other descriptor contents must remain equal,
including fields, numbers, defaults, feature definitions, options and unknown
fields. New files or further changes fail the audit and retain diagnostics.
SourceCodeInfo documentation also describes negative option-part paths in the
new compiler; its existing repeated int32 representation is unchanged.

The preview compiler runs the complete historical/frozen suite, five-generator
optional entrypoints and mappings, oneof merge policies, floating-point comparison, required-word boundaries
and the full Editions 2023/2024/2026 scalar/complex/auxiliary preparation matrix.
Temporary consumers then pin official Go v1.37.0-rc.1 for standard JSON/text
regressions, the Edition 2026 JSON and WKT oracle, and the C02 detached V2 snapshot
experiment including its frozen downstream tests. Each consumer runs default,
purego and race tests. Standard text and JSON each receive a bounded two-worker
20-second fuzz campaign (`--fuzztime` can select a local duration).

The C02 snapshot restrictions remain in force; this does not add ProtoReflect
to production generated messages. This is not a separate build of the 37.0-rc1
C++ conformance runner. Stable 36.2 conformance remains an independent gate, and
preview qualification covers the stated compiler/Go-runtime fixtures and budget.

The compiler-version helper accepts `-rcN` and historical `-rc-N` spelling.
Candidates sort numerically before the final release of the same version while
still satisfying older stable minimums. Unrecognized prerelease forms fail
instead of being treated as a stable version.
