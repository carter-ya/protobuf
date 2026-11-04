# Go feature metadata for the gogo runtime

This package provides Go 1.15-compatible bindings for Protobuf 36.2's
`google/protobuf/go_features.proto`, including `E_Go` on descriptor FeatureSet.
It uses the gogo descriptor/runtime packages and has no V2 runtime dependency.

The generated metadata contains all four official Go feature fields and their
enum values. That is a schema/descriptor claim: individual generator semantics
are tracked independently in `compat/feature-audit/inventory.json`.

Regenerate with the pinned compiler and current generator, retaining the official
source `go_package` in the raw descriptor:

```sh
protoc -I /path/to/protoc-36.2/include \
  --gogo_out=paths=source_relative,Mgoogle/protobuf/go_features.proto=github.com/gogo/protobuf/gofeatures,Mgoogle/protobuf/descriptor.proto=github.com/gogo/protobuf/protoc-gen-gogo/descriptor:/tmp/features \
  google/protobuf/go_features.proto
cp /tmp/features/google/protobuf/go_features.pb.go gofeatures/go_features.pb.go
```

`compat/check-feature-inventory.sh` compares compiled field numbers, types,
feature-support/default/retention options and enum values with the compiler's
own include tree. It rejects future changes until explicitly reviewed.
