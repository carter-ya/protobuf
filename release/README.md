# Maintenance release contract

This change prepares a candidate; it does not publish it. `manifest.json` is the machine-readable
source of truth for proposed versions, minimum toolchains, compiler and the
optional official-runtime pin. Preparing this contract does not create tags,
GitHub Releases or artifacts. The manifest records the RC channel and manual
publication policy; actual publication status is established by remote tags and
GitHub Releases, not a permanently embedded unpublished flag.

| Component | Import/module path | Proposed tag | Minimum Go |
| --- | --- | --- | --- |
| Legacy runtime and all five generators | github.com/gogo/protobuf | v1.4.0-rc.1 | 1.15 |
| Optional live reflection | github.com/gogo/protobuf/reflect | reflect/v0.1.0-rc.1 | 1.23 |

The root minor version reflects additive, opt-in APIs while retaining old
imports and unselected layouts. The new reflection module starts at 0.1: its
supported storage profile and public APIs must be stated explicitly. Both tags
must identify the same reviewed commit from `baseline/v1.3.2`. Do not retag an
existing version. Upstream v1.3.2 remains an immutable comparison baseline.

See [installation](INSTALL.md) for actual remote-consumer commands,
[the cold-cache gate and publication plan](REMOTE.md), and
[release notes](NOTES.md) for capabilities and migration boundaries.

## Official runtime decision

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
3. Prepare the two release notes and a paired tag plan. Actual publication is
   a separate action after this work is reviewed; do not create placeholder tags
   just to make an installation example pass.
4. After publishing both tags at the selected commit, rerun the same gate in
   published-version mode. Verify each tag resolves to that commit, and both
   module versions are downloadable with their checksums. Keep the RC label
   until versioned installation and intended downstream usage have passed.
5. Publish only the qualified capabilities and their minimum Go/compiler/runtime
   requirements. Do not describe the project as covering every official Go API
   or every gogo custom representation. OPAQUE and MessageSet remain deferred.

The contract and remote gate belong in the ordinary PR/Stack workflow. A passing
commit-based gate establishes remote packaging readiness; it is not evidence
that unpublished candidate tags already exist or that post-tag verification ran.
