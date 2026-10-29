# Modern conformance baseline

This gate runs the actual binary/JSON and text suites from upstream Protobuf
36.2, including proto2, proto3, the equivalent Edition 2023 schemas and the
Edition-specific DELIMITED cases. The protocol and test schemas are generated
without edits in a disposable module; they are not runtime dependencies.

Build the runner with Python 3.12, CMake, Ninja and a C++ compiler:

```
python3 compat/conformance/build-runner.py /tmp/gogo-conformance
```

The release archive is pinned by SHA-256. Dependency versions come from that
release's CMake configuration. Only the small executable entrypoint is replaced:
upstream's combined main short-circuits after a failed binary/JSON suite, which
would hide the text suite. The upstream suite implementations remain unmodified.
CI caches the executable under a key containing the entrypoint and build script.

With protoc 36.2 on PATH, run:

```
PROTOBUF_CONFORMANCE_SOURCE=/tmp/gogo-conformance/protobuf-36.2 \
PROTOBUF_CONFORMANCE_RUNNER=/tmp/gogo-conformance/bin/conformance_test_runner \
CONFORMANCE_OUTPUT=/tmp/gogo-conformance-report sh compat/check-conformance.sh
```

A missing executable is an error. Both suites enforce recommended tests too.
`report.json` separates required/recommended failures by syntax; full runner logs
are uploaded by CI. The inventories contain 5,649 binary/JSON cases and 909 text
cases. This baseline is a regression gate, not a claim of full conformance.

`failures.json` enumerates every historical stable failure with its observed
reason and follow-up area. New failures, new skips, missing tests and runner
crashes fail the gate. An improvement must remove its baseline entry.

There are two explicit exceptions, also listed by exact test name:

* Four upstream unstable-future-Edition cases are deliberately skipped. They are
  outside the supported Edition range; no stable-Edition case is skipped.
* Sixteen legacy JSON oneof/null cases can either pass or fail: the historical
  parser iterates over oneof members in random map order. The gate reports their
  actual outcome without allowing any other case to fluctuate. A03's separate
  standard JSON entrypoint will remove this ambiguity for standard-mode users;
  the legacy default is preserved for compatibility.

The binary and text failures outside A01–A04 remain explicit work for subsequent
runtime conformance upgrades. No tests are wildcarded or silently added to a
failure list during CI. The existing historical conformance directory is kept
for old workflows; this gate does not depend on it.
