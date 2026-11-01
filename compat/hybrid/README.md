# Native ProtoReflect and HYBRID qualification

`sh compat/check-hybrid.sh` generates independent consumers through gogo,
gogofast, gogofaster, gogoslick and gogotypes. Each runs default and purego:
proto2/proto3 in both native OPEN-reflection and HYBRID mode, plus Editions
2023/2024/2026 selected by scoped Go features (70 consumer runs).

The tests exercise original native type identity, direct and reflective writes,
official `prototest.Message`, legacy/modern binary cross-decoding, JSON/text,
Clone/Merge, closed enums, required fields, extensions and groups. HYBRID tests
cover presence, setters, clear/has/which, oneof case values and builder aliasing.
Editions tests include file/message/nested API inheritance and 2026 custom enum
JSON metadata through an explicit local resolver. Read-only concurrency and race
checks live in the separate live-reflection module gate.

The compiler is supplied by PATH. CI pins protoc 36.2. Preview qualification can
use protoc 37-rc1 and `--runtime-version v1.37.0-rc.1` with Go 1.27. The default
runtime revision matches reflect/go.mod; the parent module still targets 1.15.

Unselected output remains in the historical compatibility and real-project
gates. Selection normalizes convenience-generator defaults to the qualified
native storage profile; explicitly incompatible mappings produce diagnostics.
OPAQUE is outside this implementation.

`mixed.py` imports gogo and gogofaster HYBRID packages sharing option-only
metadata in one process, under the race detector. CLI entrypoints preserve
source descriptors through `command.ReadWithSources` / `GenerateWithSources`
before backend defaults are injected, so equivalent option dependencies cannot
conflict at initialization. The live module also races local resolver lookups
against cold discovery of further option extensions.
