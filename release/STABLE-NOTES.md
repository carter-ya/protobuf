# Root v1.4.1 stable with existing reflect v0.1.0-rc.2

Root v1.4.1 identifies `58ec58f77b1c65d9805d60502df5b6476f08dd4e`. The optional reflect RC2
remains at `3af6d45fd9dbe0ff1249afa246a44caf995b29a1`. The exact remote combination passed
published-version cold-cache installation on Go1.15, Go1.23 and Go1.27.
See [installation](INSTALL.md) and the independent component identities in
manifest.json. The existing v1.4.0 and RC tags were preserved.

## Patch fixes

- Optional standard time/duration/wrapper mappings preserve unknown wrong-wire
  occurrences so later fields are decoded normally.
- Standard text maps use the schema enum default when the value is omitted.
- Packed closed-enum unknown filtering avoids quadratic copies of the input
  tail and preserves subsequent fields, including extensions.
- Omitted map message values enforce required fields while preserving the
  existing nil representation for messages without required fields.

The master integration also restores historical protoc 2.6.1/3.0.2/3.14.0
consumer checks at the supported Go1.15 minimum and on Go1.23. Full maintained
suites use the pinned current compiler, which can parse Editions metadata.

## Versions and support

- Root runtime and all five generators: stable `v1.4.1`, canonical module
  `github.com/gogo/protobuf`, minimum Go1.15, no official V2 dependency.
  Remote zip sum: `h1:k443S0iIz3TDJ0wGRjvy+OAH851lhl1XzOsFD7tkfIc=`;
  go.mod sum: `h1:P1XiOD3dCwIKUDQYPy72D8LYyHL2YPYrpS2s69NZV8Q=`.
- Optional modern module: unchanged `reflect/v0.1.0-rc.2`, canonical module
  `github.com/gogo/protobuf/reflect`, minimum Go1.23 and minimum root
  `v1.4.0-rc.2`. The stable patch satisfies that minimum.
  Zip sum: `h1:L1WsD3DuZ/mO7X5G8y2IkSVLuXSNP9gkduYcwIawUSg=`;
  go.mod sum: `h1:QH+77kHBTGQyhwF0lJjcePKNNHD1CcuySLbuRU+ZtLs=`.
- The optional module retains `google.golang.org/protobuf`
  `v1.36.13-0.20261007182434-0c373ff94f29` with exact checksums in
  manifest.json. Root publication does not promote or republish that RC.

## Qualification and boundaries

Actual tag installation passed ten legacy cases on Go1.15 and 105 cases each
on Go1.23 and Go1.27, including five generators, default/purego/race,
OPEN/HYBRID/OPAQUE, protoopaque, Edition2026 and the stable-runtime negative
control. Reports verify full tag origins and remote zip/go.mod sums.
[Actual stable-tag consumer CI](https://github.com/carter-ya/protobuf/actions/runs/41822549164).

Pinned Prometheus, Kubernetes API/apimachinery, BuildKit and CockroachDB errors
packages are qualified with original/fork controls, frozen/supported regenerated
bindings, default/purego/race and bidirectional wire exchange. This is selected
package qualification, not whole-product deployment certification.

Old default APIs and layouts remain compatible. Modern native reflection,
HYBRID and OPAQUE are opt-in and require the optional RC. OPAQUE requires moving
direct field access to accessors/builders. Native pointer storage and eager
decoding remain; no memory-saving or lazy-decoding claim is added.

Legacy MessageSet remains supported; modern live reflection rejects MessageSet
and the documented non-native/custom/value/embedded representations. See
[COMPATIBILITY.md](../COMPATIBILITY.md), [modern contract](../reflect/README.md)
and [support policy](SUPPORT.md). This release does not claim every official Go
API or conformance case. Historical RC2 notes remain in NOTES.md. The earlier
v1.4.0 tag remains at `057d06490a721bea0fceb39c5c088bc1ef3085e8`; its initial
qualification is retained in [the historical tag CI](https://github.com/carter-ya/protobuf/actions/runs/41800835625).

## Optional-module source fixes after RC2

The current optional-module source fixes quadratic unknown-buffer allocation
paths and repeated whole-message validation during closed-enum decoding. The
immutable published reflect RC2 does not include these fixes and may use
disproportionate CPU or memory on wire inputs with many unknown/closed-enum
occurrences. Bound input sizes and occurrence counts when using that RC with
untrusted input, or qualify a full source revision separately. Root v1.4.1 does
not republish the nested module. See reflect/README.md for this known boundary.
