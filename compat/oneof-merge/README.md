# Opt-in oneof message merging

Generate with `--gogo_out=merge_oneof_messages=true:OUTPUT` (or the corresponding
backend's output flag) to merge consecutive occurrences of the same oneof message
member. Switching to another member clears the previous selection. The parameter
also applies to nested messages generated in that invocation. Existing bindings
and default/false generation retain historical replacement behavior.

The parameter supports native message representations with the gogo runtime.
Custom, cast and standard Go representations on oneof message fields are rejected
with a scoped diagnostic. They need separate representation-specific merging
qualification. Imported dependencies retain their own generated decoding policy;
this flag does not retrofit precompiled dependencies or custom decoding hooks.

Files that use the policy emit `oneof_merge` metadata and require runtime Version12.
Scalar-only oneofs do not acquire that guard. Generated public field and wrapper
layouts and raw schema descriptors are unchanged. The five-backend matrix covers
proto2/proto3, default/purego builds, table/direct/buffer decoding, repeated message
occurrences, field switching, wrong-wire input and nil wrappers. Both compiler
families run this gate in CI; a pre-Version12 runtime must reject opted-in code.

The official conformance profiles use separately generated binaries: legacy code
uses historical oneof semantics and the standard binary uses this parameter.
This prevents a standard-profile improvement from changing the legacy inventory.

The scalar-member qualification also covers proto2/proto3 oneofs with automatic
nullability and explicit `nullable=true`. Fast entrypoints retain wrapper-value
layout, and cloned field metadata is normalized for marshal/size, Equal/Compare
and Populate generation. Selected zero retains oneof presence. Auxiliary Clone,
VerboseEqual, ordering, Populate, String and GoString checks run in both modes.
