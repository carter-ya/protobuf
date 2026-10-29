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
release's CMake configuration. The executable entrypoint is replaced:
upstream's combined main short-circuits after a failed binary/JSON suite, which
would hide the text suite. A CMake alias makes the pinned dependency check accept
the static JsonCpp target. The upstream suite implementations remain unmodified.
CI pins Ubuntu 24.04 and caches the executable under a key containing the
platform, entrypoint and build script. JsonCpp and Protobuf are linked statically,
so the cached runner does not depend on libraries left in the build directory.
A startup probe rejects broken cache entries and rebuilds them before testing.

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

A02 removes 64 stable binary/JSON failures through compatible integer, base64
and escaped-enum parsing improvements. The remaining limitations stay explicit.

`failures.json` enumerates every remaining historical stable failure with its observed
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

A03 adds an independent standard-mode run and `failures-standard.json`. Its
inventory is the same, but no variable outcomes are permitted. The legacy
baseline remains separate so adopting standard rules does not silently change
old callers. The report directories identify the entrypoint explicitly.

B05 fixes message-oneof merging only for explicit Editions metadata. Six exact
Editions_Proto2/Proto3 binary/JSON failures are removed from each baseline.
Historical proto2/proto3 oneof behavior and text baselines are unchanged.
