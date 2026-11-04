# Go enum prefix generation

All five gogo backends implement the official Go `strip_enum_prefix` feature
for Editions 2024/2026. File settings inherit into nested enums; enum and enum
entry settings override their enclosing scope. KEEP and UNSPECIFIED retain the
original value name. STRIP removes a matching enum-name prefix while ignoring
case/underscores. A missing match or empty remainder keeps the original name.
GENERATE_BOTH emits the shorter name plus an alias for the old spelling.

Gogo's existing enum type/message prefix policy still applies. In particular,
gogoslick retains its historical choice to omit that extra Go prefix. Unmarked
legacy and default-KEEP generation retain their names. Binary numbers, proto
names, enum maps, String results, descriptors and JSON names are unchanged.
The option changes Go constant names, not the protobuf schema names.

Default constants/getters, imported references and public-import aliases use
the same resolved names. GENERATE_BOTH is the migration option when callers need
old constants to remain available. Regenerate files that reference the changed
enum as well as its declaration. Go name collisions, including those across
request files in one Go package, and invalid Go identifiers are diagnosed.
Combining stripping with custom Go enum/type value names is rejected; KEEP
continues to support those existing gogo options.

`sh compat/check-go-enum-prefix.sh` runs 100 consumers per compiler: five
backends, Editions 2024/2026, default and all four prefix enum values,
default/purego. It compiles constants, public-import aliases, nonzero first-enum
defaults and proto2 references in the same/different Go package. It checks enum
entry/enum overrides, aliases, case/underscore matching, unmatched names,
empty-remainder fallback and original binary/standard JSON names. Twenty
negative cases cover local/cross-file Go name collisions, custom-name conflicts
and invalid stripped Go identifiers. Pre-Editions protoc skips this feature gate.

The shared API consumer runs with official Go v1.36.12 for 2024 and
v1.37.0-rc.1 for 2024/2026, including the expected constant/alias sets. Modern
oracle dependencies remain isolated. The real map/public-import fixture also
covers first-use dependency validation: inspect descriptor metadata without
calling import-recording Go type emission before its per-file state exists.
A deterministic fresh-generator regression reproduces the previous panic.
