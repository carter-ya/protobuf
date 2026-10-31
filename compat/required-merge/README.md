# Final required initialization

A singular message can occur more than once on the wire. Required fields are
checked after the surviving fragments merge, including ordinary child messages,
oneof messages and DELIMITED groups. Repeated elements remain separate messages;
duplicate map entries replace earlier values, while multiple value occurrences
within one entry merge. Omitted message map values still require initialization.
Malformed wire and checked UTF-8 fail within their original occurrence, even if a
later occurrence overwrites that value.

Editions 2023/2024/2026 generation selects this policy automatically. Proto2 and
proto3 require the explicit `merge_required_fields=true` generator parameter;
use `merge_oneof_messages=true` too when adopting standard oneof merging.
Unmarked historical bindings retain their required-check timing. Regenerate
both enclosing and referenced codecs: frozen imported methods retain their own
policy. Generated files reaching required fields assert runtime Version14.
Fast decoders use the shared table path for these graphs; unrelated paths stay
generated. Direct generated decode and table entrypoints share the same policy.
`UnmarshalMerge` validates the complete resulting value, including fields present
before the call.

Every required field needs an independent nil/non-nil presence representation.
Non-nullable required scalars, singular message fields without independent
presence, and mapped message paths with required descendants
produce generation errors rather than guessing presence. Proto2 gogofaster/slick
users may need explicit `nullable=true` or retained defaults to preserve pointers.
Native bytes already distinguish nil from present empty bytes. This parameter
changes initialization timing; it does not select new JSON/text or closed-enum
policies for old schemas. Alternate runtimes are unsupported for this policy.

Run `sh compat/check-required-merge.sh` with pinned protoc 3.21.8, 36.2 or
37.0-rc1. The stable and preview matrices cover 35 profiles across five backends:
legacy controls and opt-in proto2/proto3 plus automatic Editions, each under
default and purego runtimes. Protoc 3.21.8 covers the 20 legacy-syntax profiles.
Negative tests cover unsupported presence, the alternate runtime, invalid flags
and a pre-Version14 runtime. The separate C++/official-Go binary differential
matrix verifies independently accepted ordinary required-child fragments.
