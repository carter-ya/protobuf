# Install the maintained fork

Use a specific reviewed commit until candidate tags are published. The examples
below use merged revision `5fe34bcbecd8b90338da596c7161cc7fd5e863eb` as a real,
downloadable example. That example predates the release-preparation changes;
substitute the final qualified commit when testing the candidate itself.

## Install the generators without changing imports

Create a dedicated tools module outside the repository. These commands use
remote replacements; no checkout path is referenced:

```sh
mkdir protobuf-tools
cd protobuf-tools
go mod init example.com/protobuf-tools
go mod edit -require=github.com/gogo/protobuf@v1.3.2
go mod edit -replace=github.com/gogo/protobuf=github.com/carter-ya/protobuf@5fe34bcbecd8b90338da596c7161cc7fd5e863eb
go mod download github.com/gogo/protobuf
go install github.com/gogo/protobuf/protoc-gen-gogo github.com/gogo/protobuf/protoc-gen-gogofast github.com/gogo/protobuf/protoc-gen-gogofaster github.com/gogo/protobuf/protoc-gen-gogoslick github.com/gogo/protobuf/protoc-gen-gogotypes
```

Choose `GOBIN` to keep these tools isolated, and use that directory explicitly
in your generation script. Existing applications retain canonical gogo imports.
Direct `go install github.com/carter-ya/protobuf/protoc-gen-gogo@VERSION` is
rejected because the module declares `github.com/gogo/protobuf`. Installing the
canonical path with `@VERSION` would fetch archived upstream instead. The
unversioned installation above intentionally uses the dedicated module graph.

Legacy projects need only the root replacement, and the root still supports Go
1.15. Match the runtime replacement to the generator's source revision.

## Add native reflection or HYBRID

For Go 1.23+ applications, configure both remote replacements at the same commit:

```sh
go mod edit -require=github.com/gogo/protobuf@v1.3.2
go mod edit -require=github.com/gogo/protobuf/reflect@v0.0.0
go mod edit -replace=github.com/gogo/protobuf=github.com/carter-ya/protobuf@5fe34bcbecd8b90338da596c7161cc7fd5e863eb
go mod edit -replace=github.com/gogo/protobuf/reflect=github.com/carter-ya/protobuf/reflect@5fe34bcbecd8b90338da596c7161cc7fd5e863eb
go mod download
```

Generate using the installed tool and pinned protoc 36.2:

```sh
protoc --gogo_out=paths=source_relative,default_api_level=API_HYBRID:. example.proto
# OPEN fields plus native reflection only:
protoc --gogo_out=paths=source_relative,proto_reflect=true:. example.proto
```

Editions can also select `features.(pb.go).api_level = API_HYBRID` at file or
message scope. Enable that setting only for representations the live module
qualifies. Run `go mod tidy` after adding the actual generated imports, then
`go test ./...`. Tidy may remove unused requirements before those imports exist.
Read [the native module contract](../reflect/README.md) before changing APIs.

The `/reflect` module's development-only `replace => ..` is not inherited by
applications. Replacing only `/reflect` can select old upstream root runtime and
fail the version guard. The two replacements are deliberate, even when the
application already requires upstream v1.3.2. Check `go list -m -json all` to
confirm both selected fork versions and their downloaded module directories.

## After publication

Proposed pairing: root `v1.4.0-rc.1`, submodule tag `reflect/v0.1.0-rc.1` (its
Go version string is `v0.1.0-rc.1`, without the tag prefix). Neither exists yet.
After both tags are published at the same reviewed commit, replace the two
commit queries with their respective version strings. A tag URL is not proof
that both module zips can be fetched: run the published-version consumer gate
before announcing successful installation. Retain go.sum in applications.
