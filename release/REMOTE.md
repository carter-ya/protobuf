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

The complete modern profile performs 60 tests: five remotely installed generators
through legacy, native OPEN-reflection, HYBRID and Edition 2026 consumers in
default/purego/race modes. Go 1.15 performs ten legacy default/purego tests.
Legacy fixtures preserve backend defaults; they do not silently select the
modern pointer layout to make old compatibility pass.

The gate downloads root and `/reflect` module zips from remote versions, checks
full origins where exposed by Go, independently resolves public commit/tag
metadata, and retains zip/go.mod checksums. It verifies downloaded module
paths and manifest identity, installed generator build identities, and every
consumer's selected module graph. Canonical module requirements use remote fork
replacements; all dependency sources must come from the new module cache.
Main consumer/generated files are allowed outside it. Local-path replacements,
mixed component revisions, mismatched versions and runtime overrides fail.
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

## Prepare the paired publication plan

After the final commit's complete CI and code review succeed, download its
three reports and create a reviewable plan:

```sh
python3 release/plan.py --revision FULL_QUALIFIED_COMMIT \
  --report /tmp/go115/qualification.json \
  --report /tmp/go123/qualification.json \
  --report /tmp/go127/qualification.json \
  --output /tmp/paired-release-plan.json
```

The plan rejects missing/duplicate consumer cases, mixed revisions/contracts,
changed checksums and missing toolchain/control evidence. It prints no tag or
Release API requests; the JSON contains proposed annotated-tag and atomic-push
commands for explicit maintainer review. The selected commit must be contained
in the maintenance branch, and existing tags must never be overwritten.

## Verify the published versions

Only after both paired tags have actually been published, add `--published` to
the same consumer commands. This changes module queries from commit-based
pseudo-versions to manifest versions and separately verifies the prefixed
`reflect/` tag. Every tag must resolve to the selected full commit. Missing tags
fail without creating them. Go 1.15 uses the independent tag metadata check when
its module download output does not expose Origin.Hash.

The workflow also supports a manual `published` input and automatically selects
published mode for tag pushes. A passing commit-based report is not evidence
that the proposed tags exist or that the post-tag version check succeeded.
Prepare and post-tag reports are not interchangeable in the publication plan.
