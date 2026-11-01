# Native HYBRID conformance differential

This profile generates the unmodified Protobuf 36.2 proto3 all-types schema
with native HYBRID methods and an independent official Go generator control.
Both use the same pinned official Go runtime and upstream C++ test runner.
The checked-in exact-name inventory contains 1,503 binary/JSON and 427 text
cases. Every selected case must run; new failures, skips, inventory changes,
runner failures and any native/control outcome difference fail the gate.

The current result is 1,919 successes and 11 shared official Go failures,
with zero skips and zero additional native failures. They are recorded by exact
name in `official-failures.json`: one overlong tag, three JSON deep-nesting
limits, two octal float inputs, four surrogate escape cases and one unknown-wire
text-printing case. These concern the chosen official Go codec/runtime behavior;
this gate does not relabel them as successful Protobuf conformance. Baseline
changes require explicit review even if a failure starts passing.

Proto2 all-types includes MessageSet, outside the qualified native storage
profile. It is not included here. Proto2 closed enums/required/extensions/groups
and Editions 2023/2024/2026 are tested independently in the live module and
five-generator consumer matrix. The full historical conformance gates continue
separately. This profile does not claim all schemas or OPAQUE support.

```
python3 compat/native-conformance/prepare.py UPSTREAM OUTPUT/native
python3 compat/native-conformance/prepare.py UPSTREAM OUTPUT/official --official-generator /path/to/protoc-gen-go
python3 compat/native-conformance/run.py RUNNER OUTPUT/native/testee OUTPUT/report --official-testee OUTPUT/official/testee
```

CI retains both sides' full runner logs and structured comparison report.
