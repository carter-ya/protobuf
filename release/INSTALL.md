# Install the maintained fork

The published pair is root `v1.4.0-rc.1` and `reflect/v0.1.0-rc.1`, at commit
`533267c4c68359c533a07ece14e6cb2320a22b8b`. That pair provides native reflection
and HYBRID, and does not include OPAQUE. The next RC2 manifest is a development
contract, not evidence that RC2 tags or Releases exist.

## Published tools and legacy runtime

Create a dedicated tools module outside the repository, then use remote fork
replacements while preserving canonical imports:

```sh
mkdir protobuf-tools
cd protobuf-tools
go mod init example.com/protobuf-tools
go mod edit -require=github.com/gogo/protobuf@v1.4.0-rc.1
go mod edit -replace=github.com/gogo/protobuf=github.com/carter-ya/protobuf@v1.4.0-rc.1
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
go mod edit -require=github.com/gogo/protobuf@v1.4.0-rc.1
go mod edit -require=github.com/gogo/protobuf/reflect@v0.1.0-rc.1
go mod edit -replace=github.com/gogo/protobuf=github.com/carter-ya/protobuf@v1.4.0-rc.1
go mod edit -replace=github.com/gogo/protobuf/reflect=github.com/carter-ya/protobuf/reflect@v0.1.0-rc.1
```

Use protoc 36.2 and the matching installed generator:

```sh
protoc --gogo_out=paths=source_relative,default_api_level=API_HYBRID:. example.proto
protoc --gogo_out=paths=source_relative,proto_reflect=true:. example.proto
```

## OPAQUE development and next candidate

Use the exact full source SHA from the successful PR qualification report for
both modules and the generators. `FULL_REVIEWED_COMMIT` below is a placeholder
for that SHA, not a published version:

```sh
REVISION=FULL_REVIEWED_COMMIT
go mod edit -require=github.com/gogo/protobuf@v1.4.0-rc.2
go mod edit -require=github.com/gogo/protobuf/reflect@v0.1.0-rc.2
go mod edit -replace=github.com/gogo/protobuf=github.com/carter-ya/protobuf@$REVISION
go mod edit -replace=github.com/gogo/protobuf/reflect=github.com/carter-ya/protobuf/reflect@$REVISION
```

Install the generators through a tools module with the same root replacement.
Select `default_api_level=API_OPAQUE` for private protobuf fields, or generate
HYBRID and test with `go test -tags=protoopaque ./...` to switch its emitted
alternate files. `apilevelMfile.proto=API_OPEN|API_HYBRID|API_OPAQUE` sets a file
default; explicit Editions source features and nested overrides take precedence.
Default output retains the historical OPEN API, including for Editions.

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
Keep go.sum. Only after a separately approved paired tag publication can the
commit replacements become the actual RC2 versions; published-mode cold-cache
qualification must succeed before announcing that new pairing.
