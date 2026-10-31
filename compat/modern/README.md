# Modern Go integration experiment (C02)

This isolated Go 1.23 module evaluates explicit conversion between existing gogo
messages and the official V2 runtime. It adds no dependency to the Go 1.15 root
runtime and changes no generated Go API. It is an experiment, not a released
adapter package or a guarantee that gogo values implement `ProtoReflect`.

Run `sh compat/check-modern.sh` with protoc 36.2. CI runs Go 1.23 and 1.27, default
and purego consumers. The official runtime is pinned to the same development
commit as the existing Edition 2026 oracle (`0c373ff94f29`); the stable 1.36.12
runtime predates the tested Edition 2026 JSON option support.

`NewRegistry` reads a complete descriptor set into local Files/Types registries.
Include option-only dependencies, extension declarations and all Any payloads.
Dynamic options are decoded again with local extension types before final
feature resolution. No V2 global registration is installed. Descriptor bundles
must match the actual generated runtime descriptors, including options; a
same-name schema is not sufficient. Snapshot and Apply recursively verify the
ordinary dependency closure against registered generated metadata, including
imported enum-only files and transitive dependencies. Resolved extensions also
validate their declaring files, even when those files import the message rather
than being imported by it. Any payloads validate their own source/import closure;
payload types require registered generated metadata. Missing runtime dependency
metadata is an error; documented historical gogo path aliases are recognized. Frozen WKT runtime descriptors include gogo
options that differ from compiler source options. Test helpers retain those
original descriptors and handle historical gogo import-path spellings.

`Snapshot` copies through binary into a detached dynamic message. Reflection,
protojson, local Any resolution and extension APIs operate on that snapshot.
`Apply` explicitly replaces a gogo value: it first decodes a fresh candidate,
converts it back and compares the complete modern message. A lossy conversion
returns `ErrLossyConversion` without modifying the target. Neither operation is
a live view; callers must synchronize ordinary protobuf mutation.

The checks cover:

* Optional synthetic oneofs, absent/zero/empty, unknown fields, oneof values,
  detached mutation, local message/URL lookup and explicit replacement.
* Editions 2023 DELIMITED fields, enum openness and extensions (including nested
  declaration scope), 2024 option imports, and 2026 custom enum JSON names.
* All five optional backends, scalar/enum/bytes casts and custom/std mappings,
  default and purego builds, independent protojson parity and transactionally
  rejected inner WKT unknown-field loss.
* Actual frozen etcd raft v3.5.0 and CockroachDB errors v1.8.1 bindings, including
  error detail payloads through Any, protojson and subsequent error decoding.
* Missing required fields/dependencies, malformed or mismatched descriptors,
  nil targets, incompatible target descriptors and disabled unknown storage.

## Confirmed boundaries

The official dynamic decoder exposes an unknown number of a CLOSED enum as a
known field. Gogo correctly retains it as unknown bytes for Editions. A binary
roundtrip alone does not reveal this reflection mismatch. This experiment
rejects such messages with `ErrUnsupportedSemantics`, including nested/list/map
and extension values, as well as values packed inside Any. Cyclic mutable
snapshots and excessively nested messages are rejected. Ordinary known closed values and open-enum unknowns are
supported. It does not rewrite descriptors to turn closed enums into open ones.

The official default build also rejects MessageSet descriptors. That error is
preserved and tested explicitly; no legacy feature build tag is silently added.
Missing Any payload types are errors, and mappings unable to represent nested
unknown fields, presence or ranges cannot be applied without an error.

These results support an explicit conversion approach for further adapter work.
They do not justify automatically wrapping every historical gogo message with
the official legacy adapter, or claiming transparent V2 reflection for all gogo
representations. A production adapter requires a separate API and support review.
