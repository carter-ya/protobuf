# Real-project compatibility matrix

This matrix exercises actual checksum-pinned upstream modules. It does not copy
simplified versions of their generated types or silently modify application code.
The old etcd raft / errors v1.8.1 suite remains an independent frozen gate.

| Profile | Fixed modules | Upstream packages | Generation coverage |
| --- | --- | --- | --- |
| Prometheus | prometheus v0.315.1 (release 3.15.1) | prompb, remote-write v2, client and rwcommon | Frozen plus original/current gogofast regeneration |
| Kubernetes | api / apimachinery v0.31.4 | core/apps API, resource.Quantity, IntOrString, metadata and protobuf serializer | Frozen go-to-protobuf bindings |
| BuildKit | buildkit v0.12.5 | solver/pb and util/apicaps | Frozen plus original/current gogofaster regeneration |
| CockroachDB | errors v1.14.0 | All errors module packages, including gRPC/error details | Frozen error and Any bindings |

Primary source evidence: [Prometheus generator configuration](https://github.com/prometheus/prometheus/blob/v3.15.1/prompb/buf.gen.yaml),
[Kubernetes dependency](https://github.com/kubernetes/apimachinery/blob/v0.31.4/go.mod),
[BuildKit generator configuration](https://github.com/moby/buildkit/blob/v0.12.5/solver/pb/generate.go),
[CockroachDB dependency](https://github.com/cockroachdb/errors/blob/v1.14.0/go.mod).
Their generated files import the original gogo runtime. Prometheus and errors are
current releases as checked on 2026-10-31. Kubernetes 0.37.2 no longer declares
that runtime dependency, while retaining a modified generator lineage; BuildKit
0.34.1 uses official Go output for solver/pb. Those versions are not counted as
original-gogo replacement coverage. Historical versions are labelled explicitly.

## Run and acceptance criteria

With Python 3.12+, Go 1.26+ and pinned protoc 36.2:

```sh
python3 compat/downstream/projects/run.py /tmp/gogo-projects
python3 compat/downstream/projects/run.py /tmp/gogo-prometheus --profile prometheus
```

The output directory must be new. `--frozen-only` limits a diagnostic run to
frozen bindings; the CI profiles use the complete configured scope. All four
profiles run default, purego and race consumers against original v1.3.2 sources
and the maintained runtime. Upstream selected-package tests run under every
variant. Original/current regeneration uses identical compiler, includes, source
files and upstream options. Go syntax and decoded descriptors must match before
testing the regenerated project. Kubernetes' go-to-protobuf source transforms
and CockroachDB's regeneration pipeline are not replaced by a vanilla invocation;
no regeneration qualification is claimed for those two profiles.

Each variant tests actual message marshal/unmarshal, Size, Clone and Merge. JSON
uses the project's real entrypoint: legacy jsonpb for Prometheus, BuildKit and
errors, and encoding/json with Kubernetes' native hooks for Kubernetes objects.
Kubernetes' bespoke resource/time mappings do not establish generic jsonpb
compatibility; the original JSONPB path already fails on this fixture. The native
Kubernetes semantic equality implementation handles its custom Go types. A
separate regression verifies the maintained proto.Equal fix on real UID metadata.

Every fork variant decodes original-runtime message bytes. Every original
variant independently decodes fork bytes. Normalized JSON and wire lengths must
match; wire byte order need not match because Protobuf is not canonical. The
original Buffer deterministic path rejects some generated Marshal methods, so
it is not used as a false oracle for deterministic encoding.

Two known original defects are explicit controls: Equal panics on Kubernetes
UID values in every mode; purego Clone/Merge panics on named fields. The original
controls assert the known panic, while maintained variants must execute the
operations successfully. They are not reported as supported original behavior.
The follow-up runtime fixes also preserve native time values without traversing
private state or mixing two timestamps' fields.

The report retains source revision/dirty state, compiler/Go versions, module and
manifest checksums, fixture/harness hashes, every command's log and tested modes.
Original production sources come from immutable tag b03c65ea; only development
go.mod/go.sum files are normalized to build with the current compiler. Root
runtime requirements remain Go 1.15. Newer consumer requirements are isolated.
CI retains reports, logs, module graphs and message records. These are targeted
package suites, not Kubernetes cluster, Docker daemon or whole-product system
integration tests.

## Local qualification

The 2026-10-31 run passed 68 checks: Prometheus 27, Kubernetes 9, BuildKit 23 and
errors 9. It includes eight real message fixtures, both runtime implementations,
all three build variants and two regenerated projects. The maintained production
source is PR #63 head ad6fa08b; compiler is 36.2 and Go is 1.27.0. Full remote CI
remains required for each submitted PR head and future changes.

## Live reflection boundaries

Maintained-runtime consumers additionally test the optional live reflection
module. Actual Prometheus Sample/Label storage qualifies and runs live writes,
modern JSON/binary interoperability and official `prototest.Message` in each
variant. The existing complex project fixtures retain their original Go
representations: flattened/value messages, custom mappings and missing unknown
storage are rejected explicitly rather than silently copied into dynamic types.
The runtime supplies the gogo option descriptor itself, so Kubernetes packages
that omit that Go import still reach the precise storage diagnostic.

These checks are distinct from native HYBRID regeneration. Upstream schema
options are not rewritten to make a migration appear transparent. The full
68-check original/replacement matrix remains required, including frozen and
regenerated consumers. New reflection tests run only against the maintained
runtime; original controls retain their original dependency graph.
