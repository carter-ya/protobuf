# Install the maintained fork

The published pair is root [v1.4.0-rc.2](https://github.com/carter-ya/protobuf/releases/tag/v1.4.0-rc.2)
and [reflect/v0.1.0-rc.2](https://github.com/carter-ya/protobuf/releases/tag/reflect/v0.1.0-rc.2),
at commit `3af6d45fd9dbe0ff1249afa246a44caf995b29a1`. It provides native
reflection, HYBRID, OPAQUE and `protoopaque` builds. RC1 remains available as a
historical pair without OPAQUE. Both RC2 tags passed published-version
cold-cache installation; [release notes](NOTES.md) link the completed workflows.

## Published tools and legacy runtime

Create a dedicated tools module outside the repository, then use remote fork
replacements while preserving canonical imports:

```sh
mkdir protobuf-tools
cd protobuf-tools
go mod init example.com/protobuf-tools
go mod edit -require=github.com/gogo/protobuf@v1.4.0-rc.2
go mod edit -replace=github.com/gogo/protobuf=github.com/carter-ya/protobuf@v1.4.0-rc.2
go install -mod=mod github.com/gogo/protobuf/protoc-gen-gogo github.com/gogo/protobuf/protoc-gen-gogofast github.com/gogo/protobuf/protoc-gen-gogofaster github.com/gogo/protobuf/protoc-gen-gogoslick github.com/gogo/protobuf/protoc-gen-gogotypes
```

Choose an isolated `GOBIN` and use it explicitly in generation scripts. Legacy
projects need only the root replacement and retain Go 1.15 support. Match the
runtime and generator source versions. Direct fork-path versioned `go install`
conflicts with the preserved module declaration; canonical `go install ...@VERSION`
fetches archived upstream. The unversioned command above uses the tools graph.

## Published native reflection and HYBRID

For Go 1.23+ applications, add both replacements:

```sh
go mod edit -require=github.com/gogo/protobuf@v1.4.0-rc.2
go mod edit -require=github.com/gogo/protobuf/reflect@v0.1.0-rc.2
go mod edit -replace=github.com/gogo/protobuf=github.com/carter-ya/protobuf@v1.4.0-rc.2
go mod edit -replace=github.com/gogo/protobuf/reflect=github.com/carter-ya/protobuf/reflect@v0.1.0-rc.2
```

Use protoc 36.2 and the matching installed generator:

```sh
protoc --gogo_out=paths=source_relative,default_api_level=API_HYBRID:. example.proto
protoc --gogo_out=paths=source_relative,proto_reflect=true:. example.proto
```

## Published OPAQUE and protoopaque builds

Install the generators through the tools module above and add both RC2 runtime
replacements. Select private protobuf fields explicitly:

```sh
protoc --gogo_out=paths=source_relative,default_api_level=API_OPAQUE:. example.proto
```

Alternatively generate HYBRID with the earlier command and switch its mutually
exclusive generated files using `go test -tags=protoopaque ./...`.
`apilevelMfile.proto=API_OPEN|API_HYBRID|API_OPAQUE` sets a file default;
explicit Editions source features and nested overrides take precedence.
Default generation retains the historical OPEN API, including for Editions.

OPAQUE cannot compile existing direct field access; migrate those usages to
getters, setters, presence/clear methods and builders. Numeric scalar builder
inputs are copied; string pointers and message/slice/map values retain the
qualified official ownership semantics. Native pointer storage and eager
decoding remain; memory optimization and lazy decoding are not claimed.

New optional-module builds require root runtime version 18. The module's local
`replace => ..` is not inherited by applications: replacing only `/reflect`
can select an incompatible root. Check `go list -m -json all` for both remote
fork replacements. Read [the representation contract](../reflect/README.md)
before opting in; custom/std/embedded/value mappings and modern MessageSet
remain excluded. Legacy MessageSet and default old APIs remain available.

After adding generated imports, run `go mod tidy` and the application's tests.
Keep go.sum. For paired development after RC2, replace both fork modules with the same
reviewed full commit SHA and install its matching generators; that checkout is
not a new published version. Future tags require separate approval and their
own published-mode cold-cache verification. Existing tags must not move.


The S04 contract separately prepares root `v1.4.0` with the existing reflect
RC2. That root tag is not published by the preparation work. Use the current
published commands above until versioned installation is verified. Candidate
maintainers use [the cross-revision remote gate](REMOTE.md#candidate-root-with-published-reflect-rc2),
which selects root by full public commit and keeps reflect at its published
version; it never moves the RC2 tag or changes its minimum-root declaration.
