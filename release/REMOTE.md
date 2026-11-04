# Cold-cache public consumer gate

Run from the reviewed candidate checkout with Python 3.12, pinned protoc 36.2,
a C compiler for race tests, and the selected Go toolchain. The exact commit
must already be reachable in the public fork; an unpublished local commit cannot
qualify remote installation. Output directories must be new and outside this
checkout. Each run starts its own empty module and build caches.

```sh
python3 release/remote.py /tmp/release-go123 --revision FULL_PUBLIC_COMMIT --probe-stable
# With Go 1.27 on PATH:
python3 release/remote.py /tmp/release-go127 --revision FULL_PUBLIC_COMMIT --probe-stable
# With Go 1.15.15 on PATH:
python3 release/remote.py /tmp/release-go115 --revision FULL_PUBLIC_COMMIT --legacy-only
```

The complete modern profile performs 105 tests: five remotely installed generators
through legacy, native OPEN-reflection, HYBRID, explicit OPAQUE and Edition
2026 consumers in default/purego/race modes, plus `protoopaque` variants of
HYBRID and Edition 2026. The report profile `native-opaque-v1` and full case
inventory are mandatory; the old 60-case RC1 reports cannot qualify RC2. Go 1.15 performs ten legacy default/purego tests.
Legacy fixtures preserve backend defaults; they do not silently select the
modern pointer layout to make old compatibility pass.

The gate downloads root and `/reflect` module zips from remote versions, checks
full origins where exposed by Go, independently resolves public commit/tag
metadata, and retains zip/go.mod checksums. It verifies downloaded module
paths and manifest identity, installed generator build identities, and every
consumer's selected module graph. Canonical module requirements use remote fork
replacements; all dependency sources must come from the new module cache.
Main consumer/generated files are allowed outside it. Local-path replacements,
component revisions that differ from the explicitly selected per-component
origins, mismatched versions and runtime overrides fail. In
`root-stable-modern-rc` mode the root comes from the candidate commit and reflect
comes from the immutable published RC2 version, checksum and tag origin.
No repository go.sum/cache is copied into the consumers. GOWORK and user Go
settings are disabled; consumer Go/git subprocesses have tokens/SSH access
removed. An optional GITHUB_TOKEN is used only by the metadata API lookup.

The explicit stable control replaces the official snapshot with fixed v1.36.12.
Only the manifest's exact unsupported-Edition-2026 diagnostic is accepted as
that expected negative control. Arbitrary compile/network/test errors do not
qualify it. All positive consumer cases must pass. The snapshot checksum and
selected runtime version must match the release contract.

`.github/workflows/release-readiness.yml` runs the gate on Go 1.15.15, 1.23 and
1.27, checking out the actual PR head rather than a synthetic merge ref that
module proxies may not fetch. Artifacts retain reports/logs, module graphs,
go.mod/go.sum and generator build information; module/build caches are not
uploaded. Other full compatibility/conformance gates remain required.

## Prepare the component publication plan

After the final commit's complete CI and code review succeed, download its
three reports and create a reviewable plan:

```sh
python3 release/plan.py --revision FULL_QUALIFIED_COMMIT \
  --report /tmp/go115/qualification.json \
  --report /tmp/go123/qualification.json \
  --report /tmp/go127/qualification.json \
  --output /tmp/component-release-plan.json
```

The plan rejects missing/duplicate consumer cases, mixed revisions/contracts,
changed checksums and missing toolchain/control evidence. It prints no tag or
Release API requests; the JSON contains proposed annotated-tag and atomic-push
commands for explicit maintainer review. The selected root commit must be contained
in the maintenance branch, and existing tags must never be overwritten. The
independent-root plan contains only the new root tag and separately records
the reused reflect RC2 tag, revision and checksums. `STABLE-NOTES.md` records the root stable release; `NOTES.md` remains the
published RC2 history.

## Verify the published versions

Only after the proposed tags have actually been published, add `--published` to
the same consumer commands. This changes module queries from commit-based
pseudo-versions to manifest versions and separately verifies the prefixed
`reflect/` tag. Every tag must resolve to its selected component commit. Root stable promotion
reuses the existing reflect tag at its original commit. Missing tags
fail without creating them. Go 1.15 uses the independent tag metadata check when
its module download output does not expose Origin.Hash.

The workflow also supports a manual `published` input and automatically selects
published mode for tag pushes. A passing commit-based report is not evidence
that the proposed tags exist or that the post-tag version check succeeded.
Prepare and post-tag reports are not interchangeable in the publication plan.

The repository's default branch may not yet contain this maintenance workflow,
which can limit the GitHub UI's manual-dispatch entry. Tag pushes run the saved
workflow; maintainers can also run the published-mode CLI directly. Do not
change the default branch or weaken the gate to work around that UI limitation.

## Real downstream use of published tags

The synthetic cold-cache consumer matrix above verifies release packaging and
the native API profiles. The separate [published real-project gate](../compat/downstream/projects/README.md#published-pair-qualification)
tests the recorded immutable release pair against pinned Prometheus,
Kubernetes, BuildKit and CockroachDB errors packages. It covers original/fork
controls, supported regeneration, default/purego/race and bidirectional wire
exchange. Its fork modules, generators and includes come from the published
remote versions, while the qualification harness is reviewed independently.
The source-checkout real-project CI remains a separate gate for later code.


## Candidate root with published reflect RC2

S04's additional gate tests the proposed stable root before creating its tag:

```sh
python3 compat/downstream/projects/run.py /tmp/root-stable-projects \
  --candidate-root FULL_PUBLIC_COMMIT
```

Use Go 1.27, Python 3.12 and protoc 36.2. Optional `--candidate-repository github.com/OWNER/protobuf` selects a public fork for the root only; the reused
reflect component stays at its recorded publisher. This mode is mutually
exclusive with `--published`. The canonical root requirement is the current manifest version,
replaced remotely with the candidate's immutable resolved module version; reflect uses
the actual `v0.1.0-rc.2` replacement. Candidate mode alone does not establish that a proposed version is published.

`.github/workflows/root-stable-readiness.yml` runs all four pinned projects at
the actual PR head with empty caches. The scope remains original/fork controls,
frozen and supported regenerated packages, default/purego/race and bidirectional
wire exchange. Reports record the root candidate revision, frozen reflect tag
origin, exact downloaded module identities, generator build identity and all
module graphs. A stale root, moved reflect tag, changed checksum or unqualified
local dependency fails. These are real selected-package tests, not a claim that
the entire upstream products have passed.

The ordinary source-checkout and published-RC2 matrices also remain required.
After actual root publication, verify the stable tag with `release/remote.py --published`, then update the published record and rerun the real-project
`--published` matrix for the new immutable pair before announcing it.
