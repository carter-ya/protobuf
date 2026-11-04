# Maintenance release contract

The stable root [v1.4.0](https://github.com/carter-ya/protobuf/releases/tag/v1.4.0)
identifies `057d06490a721bea0fceb39c5c088bc1ef3085e8`. It is qualified with the existing
optional [reflect RC2](https://github.com/carter-ya/protobuf/releases/tag/reflect/v0.1.0-rc.2)
at `3af6d45fd9dbe0ff1249afa246a44caf995b29a1`. Root is stable; the optional modern
module remains RC with its immutable official-runtime snapshot. Both component
tags and their zip/go.mod checksums passed actual published-version cold-cache
installation on Go1.15/1.23/1.27. Existing RC1/RC2 tags remain immutable.

`manifest.json` separates the current source qualification contract from
`published_pair`, which records the verified tags, per-component revisions and
remote module checksums. Its legacy `revision` alias identifies the root only.
Later maintenance commits are not additional published versions. Publication
remains manual; remote tags and GitHub Releases establish availability.

| Component | Import/module path | Published tag | Minimum Go |
| --- | --- | --- | --- |
| Runtime and all five generators | github.com/gogo/protobuf | v1.4.0 (stable) | 1.15 |
| Optional live reflection | github.com/gogo/protobuf/reflect | reflect/v0.1.0-rc.2 | 1.23 |

The root minor version adds opt-in APIs while retaining old imports/default
layouts. The optional module's supported storage profile and RC status remain
explicit. A separately released root may reuse an existing optional module only
after that exact remote combination is qualified with independent provenance.
New source revisions must be contained in the manifest's maintenance trunk,
now `master`. Upstream v1.3.2 remains an immutable comparison baseline.

See [installation](INSTALL.md), [the cold-cache gate and publication plan](REMOTE.md),
[stable notes](STABLE-NOTES.md), and [historical RC2 notes](NOTES.md).

## Official runtime decision

The [root/optional-module support decision](SUPPORT.md) records the independent
root-stable route while preserving the Go1.23 modern RC line. Its machine policy
is checked in CI; a future upstream Go1.26 requirement does not silently change
this line's compatibility floor.

The optional reflection module retains
`google.golang.org/protobuf v1.36.13-0.20261007182434-0c373ff94f29` for this RC.
Its module and go.mod checksums are pinned in `manifest.json` and reflect/go.sum.
The root runtime/generators do not gain this dependency or a newer Go requirement.

The checked stable reference is v1.36.12; stable protoc is 36.2. The development
revision is required for the qualified Edition 2026 descriptor and custom enum
JSON metadata. The cold-cache stable control using v1.36.12 returns
`edition EDITION_2026 not yet supported by the Go Protobuf runtime` before
JSON decoding; this exact capability gap is recorded in the manifest.
Replacing it with the lower stable reference must not silently remove that
implemented feature. The release remote-consumer gate exercises that metadata,
not just message compilation. Go 1.37-rc previews remain separate qualification
inputs, not production dependency upgrades.

This is an explicit **RC dependency decision**, not a claim that the snapshot
is an upstream supported stable release. The pin is immutable; no `@latest`
dependency or automatic override is allowed. An application's module selection
may choose a newer official runtime: only the exact pin and separately reported
preview matrix are qualified. Supporting another runtime requires the same
reflection, codec, metadata, downstream and conformance gates.

Promoting the modern module to a stable release requires reviewing a stable
upstream runtime that contains the required metadata behavior, then passing the
full matrix at that exact version. If no such stable version is available,
keep the modern module prerelease or explicitly qualify a different policy;
do not downgrade features to obtain a green release. The legacy root can be
released independently only with release notes stating the modern module's
prerelease status and exact compatible pairing.

Primary references: [official Go module rules](https://go.dev/ref/mod),
[official runtime releases](https://github.com/protocolbuffers/protobuf-go/releases),
[upstream support policy](https://protobuf.dev/support/version-support/).

## Promotion and publication

1. Select the final immutable reviewed commit. Verify review threads are resolved
   and every applicable CI check belongs to that commit and succeeded. Check
   the root/reflect module directives, minimum versions, exact dependency sums,
   and proposed tag names against the manifest.
2. Run the remote consumer gate using that commit, in an empty module cache and
   external directory. Downloads must come from the public fork. Install all
   five generators through canonical imports, regenerate external schemas and
   exercise legacy and native V2 consumers. Retain module graphs, zip sums,
   build identities and logs. A local-path replacement cannot qualify this gate.
3. Prepare component release notes and a tag plan. In `root-stable-modern-rc`
   mode the plan creates only the selected root tag; reflect RC2 is reused at its recorded
   source and checksums. Qualify the candidate root with that published reflect
   in the four-project [remote downstream matrix](REMOTE.md). Actual publication is
   a separate action after this work is reviewed; do not create placeholder tags
   just to make an installation example pass.
4. After publishing the proposed tags, rerun the same gate in published-version
   mode. Verify each tag resolves to its selected component revision, and both
   module versions are downloadable with their checksums. The optional module
   retains its RC label. Then update `published_pair` with the verified root and
   reflect identities and per-component revisions; its legacy `revision` alias
   must equal the root revision. That record update never moves existing tags.
5. Publish only the qualified capabilities and their minimum Go/compiler/runtime
   requirements. Do not describe the project as covering every official Go API
   or every gogo custom representation. OPAQUE provides private protobuf fields
   and the qualified public accessors/builders; numeric scalar builder inputs
   are copied, while string-pointer and message/slice/map ownership follow the
   official contract. Native pointer storage and eager decoding remain in use;
   no memory-saving or lazy-decoding performance claim is made. Legacy
   MessageSet remains supported; modern live reflection rejects it. See the
   [demand assessment](../compat/downstream/projects/messageset-demand.json).

The contract and remote gate belong in the ordinary PR/Stack workflow. A passing
commit-based gate establishes remote packaging readiness; it is not evidence
that unpublished candidate tags already exist or that post-tag verification ran.


The current source candidate prepares root `v1.4.1` to fix two reviewed runtime
boundaries: preserve unknown wrong-wire occurrences of optional standard
mappings, and initialize omitted standard-text map enum values from their schema
default, avoid quadratic copying when filtering packed enums, and validate
required fields in omitted map message values. The v1.4.0 tag and all RC tags
remain immutable. This candidate is not a
new published version until its exact source and actual tag pass qualification.
