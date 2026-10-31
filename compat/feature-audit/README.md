# Exhaustive feature inventory

Reference checked on 2026-10-31: stable Protobuf 36.2 and official Go 1.36.12;
37.0-rc1 / Go 1.37.0-rc.1 remain separate previews. Sources are the official
[releases](https://github.com/protocolbuffers/protobuf/releases),
[feature settings](https://protobuf.dev/editions/features/),
[enum specification](https://protobuf.dev/programming-guides/enum/) and the
compiler's bundled `descriptor.proto` / `go_features.proto` schemas.

`inventory.json` names every core and Go feature field and every enum value.
`sh compat/check-feature-inventory.sh` reads schemas from the pinned compiler's
own include directory, compares compiled core and Go metadata, and rejects added,
removed or renumbered feature fields/values. The report distinguishes a complete
inventory from complete implementation. Known gaps remain visible and cannot be
turned into an all-features claim by a passing inventory/conformance check.
`FEATURE_AUDIT_REPORT` retains the report at a caller-selected path.

| Core feature | Responsible implementation | Existing evidence |
| --- | --- | --- |
| field_presence | Runtime/generation: implicit, explicit, required, optional synthetic oneofs | optional, fast Editions, wide required and final required-merge matrices |
| enum_type | Definition-scoped Editions filtering and unknown retention | closed/open, packed/list/map/oneof/extension binary differential cases |
| repeated_field_encoding | Runtime accepts both forms; generation selects packed/expanded | all five Editions backends |
| utf8_validation | Per-occurrence VERIFY/NONE, including maps/extensions | binary oracle and generated codec matrices |
| message_encoding | LENGTH_PREFIXED/DELIMITED and matching group boundaries | scalar/complex Editions and binary matrices |
| json_format | Protoc validates ALLOW schemas; explicit standard JSON implements the mapping | independent JSON differential and complete standard profile |
| enforce_naming_style | Protoc validates naming; raw descriptors retain settings | resolved defaults and compiler tests |
| default_symbol_visibility | Protoc validates cross-file visibility; no Go visibility rewrite is specified | exported/imported Edition 2026 fixtures |
| enforce_proto_limits | Protoc enforces schema-size limits; runtime wire bounds are separate | defaults and strict binary envelope tests |

The last three are compiler responsibilities. Descriptor retention alone is not
runtime semantic qualification. Frontend boundary tests and their generator
handoff are tracked alongside the runtime tests.

## Remaining differences at this baseline

Legacy proto2 closed enums are now available through explicit
[`enforce_closed_enums=true`](../legacy-enums/README.md) regeneration, including
imports, packed values, map entries, oneofs and extensions. Unmarked/frozen
bindings retain historical decoding. The remaining implementation gaps are:

The [legacy enum JSON API](../go-enum-json/README.md) now implements its official
false default and file/enum overrides, with independent official-generator
method and behavior checks. The remaining Go generation gaps are:

* Non-OPEN `api_level`, non-KEEP `strip_enum_prefix` and `optimize_mode` are
  diagnosed rather than implemented.
  Their individual fields/values remain in the inventory as gaps. Core Editions
  support is not equivalent to support for all official Go generator APIs.
* The C02 V2 conversion experiment is not a production ProtoReflect adapter and
  rejects documented lossy/unsupported cases. A production integration contract
  remains separate from the old Go 1.15 runtime and public struct layouts.
* Historical JSON/text entrypoints intentionally preserve legacy behavior;
  standard entrypoints are explicit. Unsupported gogo representations must be
  diagnosed or individually qualified, not silently included in a blanket claim.

The four `EditionUnstable` skips are outside the released Edition 2023/2024/2026
range. C++/Java/C#/Rust-specific API features are not Go output APIs; their schema
options must survive descriptors, but this Go backend does not generate those
languages. A future compiler/schema update requires reviewing every inventory
change before updating the reference pins.

## Downstream qualification plan

Use isolated, checksum-pinned consumers: current Prometheus remote-write types
and CockroachDB errors, plus historical Kubernetes and BuildKit releases that
still generate/import gogo types. Modern Kubernetes/BuildKit no longer having
the original dependency is recorded rather than misreported as gogo coverage.
For each selected version, run original-runtime controls and fork replacements,
then compare regenerated output against the original generator with identical
protoc/options. Exercise actual upstream tests and binary/JSON/Clone/Merge/size
paths, default/purego/race, and both frozen and regenerated bindings. Targeted
package results do not imply that infrastructure-dependent whole-project suites
or every historical release were tested.
