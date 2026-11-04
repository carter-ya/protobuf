# Generated floating-point comparison

Fresh native float/double `Compare` methods use the same ordering as
[Go sort.Float64Slice](https://pkg.go.dev/sort#Float64Slice.Less): NaNs precede
numbers and tie with other NaNs. Signed zeros tie. Equal NaN elements continue
to the next element, map entry, field or unknown bytes. This repairs the original
non-reflexive result without changing finite ordering, presence, field order or
collection-length ordering.

Run `sh compat/check-compare-floats.sh` with pinned protoc 3.21.8, 36.2 or
37.0-rc1. The gate enables Compare explicitly for all five production backends.
It covers proto2/proto3, plus Editions 2023/2024/2026 on modern compilers, in
default and purego builds. Tests compare all value pairs against the independent
Go sort order, including different NaN payloads/signs, infinities, signed zeros,
float32 rounding, nullable/implicit/optional fields, repeated fields, map values,
oneof scalars, presence and continuation after a NaN tie.

The maintained DoubleValue/FloatValue methods are refreshed from the generator;
only those two function bodies change. Freshly generated WKT wrappers repeat the
same behavioral tests. Public layouts and embedded descriptors are unchanged.
Existing external generated methods require regeneration to acquire this fix.
Custom Compare hooks retain responsibility for their own ordering.

This changes comparison ordering, not the historical floating-point equality
contract in `proto.Equal` or generated Equal/VerboseEqual methods. Their documented
Go NaN behavior is retained. Compare concerns values of the same concrete message
type; it is not a new heterogeneous-message ordering API.
