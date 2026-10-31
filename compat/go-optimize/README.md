# Go optimization strategies without changing the legacy default profile

`features.(pb.go).optimize_mode` selects an explicit safe Go codec strategy in
Editions 2024/2026. File settings inherit into messages and nested messages;
message settings override their enclosing scope. The option affects generated
code, not field layouts, field numbers, binary data or JSON/text mappings.

* SPEED generates the existing safe fast Marshal/Size/Unmarshal methods.
* CODE_SIZE generates compact wrappers over the shared runtime tables. It keeps
  those public methods available so a fast parent can call a compact child.
* UNSPECIFIED or no setting retains the chosen gogo backend's established
  strategy. In particular, choosing gogofast/faster/slick/types remains an
  explicit backend choice; it is not silently changed to reflection.

The backend remains gogo's Open Struct API. This does not implement HYBRID or
OPAQUE, transparent ProtoReflect, or the exact source output of Google's Go
backend. Production runtime dependencies and Go 1.15 remain unchanged. The
source schema retains its original feature settings in raw descriptors.

A selected SPEED parent that reaches an unselected dependency without matching
fast methods uses shared tables for that parent instead of generating references
to methods that do not exist. Mixed SPEED/CODE_SIZE graphs therefore compile
and retain native oneof/map/repeated/extension semantics. Existing unselected
partial fast-codec configurations keep their established diagnostics. Regenerate
referenced/enclosing codecs together; frozen dependencies keep their own policy.

Explicitly disabling a required Marshal/Size/Unmarshal method conflicts with a
selected strategy and is diagnosed. Protosizer combinations are outside this
profile. Existing Editions restrictions on unsafe codecs and custom field
representations still apply. No new Go layout is introduced for wrappers.

Fresh and mutated messages run a sizing pass before direct table marshaling,
including both public Marshal and XXX_Marshal entrypoints, so nested length
prefixes never depend on an absent or stale sizing pass from another caller.

Compact wrappers assert runtime Version16 and expose `XXX_TableCodec`, an
internal capability marker. Only marked generated types bypass their own custom
Marshal shortcut when entering the shared tables; this avoids recursion through
Marshal/Size. Ordinary custom marshalers retain their historical dispatch.
Oneof helper methods use the owning message's oneof metadata, emit only the
selected member, and do not require unrelated owner fields. Required fields
inside the selected payload still observe their normal validation. Direct forward and
backward buffer methods preserve their offset conventions and reject short
buffers; the compact XXX_Marshal path appends to existing data and supports its
deterministic argument.

`sh compat/check-go-optimize.sh` qualifies five backends, Editions 2024/2026,
absent/UNSPECIFIED/SPEED/CODE_SIZE file settings and default/purego builds (80
consumer runs). It tests nested overrides, mixed graphs, direct and ordinary
entrypoints, map/oneof/list/extensions, Clone, unknown retention, fragment merges,
fresh and mutated direct marshaling, forward/backward buffers, append prefixes,
deterministic table encoding,
conflicting source options and an old-runtime compile-negative guard. Generated
source must be smaller for CODE_SIZE than SPEED on the retained dense fixture;
this is a source-size assertion, not a hardware-independent speed promise.

The existing binary differential harness additionally runs both optimization
modes against the pinned C++ oracle under all five backends, both build modes
and both extension stores (20 fuzz campaigns per selected strategy). Its local
closed fixture is generated as Edition 2024 with STYLE_LEGACY so the Go feature
is legal while preserving the native C++ 2023 fixture's wire semantics. The
upstream all-types Edition 2023 fixture remains an unselected control. Reports
retain the chosen mode and source state. Default/unselected conformance and
historical consumers remain separate gates.

The pinned official Go v1.36.12/v1.37.0-rc.1 generated Go feature metadata does
not yet contain this fourth field, although the compiler's schema does. We use
the compiler schema and independent C++ wire oracle, rather than claiming an
official Go optimization behavior comparison. API-level and production V2 gaps
remain visible in the feature inventory.
