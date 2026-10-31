# Reflection map metadata cache

Matched five-sample, 200 ms benchmarks on Apple M5 Max, go version go1.27.0 darwin/arm64, libprotoc 36.2. Before production source: `6fe392e3ce24d0cdc7dd0a8a1fbfd3d1d9e729b9` (archived without production edits); after: `6816daa8787f6ab422f9d3084f03e83b7a62e842`. The same checked-out harness, templates and fixture were used, with one CPU. Source and harness identity are recorded separately.

Allocation profiling identified `Properties.Parse(strings.Join(valTags, ","))` inside each map-entry decode. Both strings and the split-tag slice were allocated for every entry, even when the value was not an enum. The cache now parses the immutable tag and enum default once when constructing the field codec. Each decoded entry still receives its own key/value and default value.

| Payload bytes | Operation | Before ns/op | After ns/op | Before allocs/op | After allocs/op | Before B/op | After B/op |
| ---: | --- | ---: | ---: | ---: | ---: | ---: | ---: |
| 64 | Encode | 1268 | 1173 | 67 | 67 | 1408 | 1408 |
| 64 | Decode | 1567 | 854.1 | 57 | 41 | 1680 | 976 |
| 4096 | Encode | 1438 | 1436 | 67 | 67 | 6080 | 6080 |
| 4096 | Decode | 1715 | 1084 | 57 | 41 | 5712 | 5008 |

With eight map entries, decoding removes exactly 16 allocations and 704 allocated bytes per operation in this fixture. Generated code bytes and wire lengths are unchanged. The timing medians are local observations, not a hardware-independent speed guarantee; encode is outside the modified path. The original [D01 baseline](BASELINE.md) remains historical data.

Evidence: [before](baseline/2026-10-31-map-cache/before) and [after](baseline/2026-10-31-map-cache/after) contain raw samples, environment and CSV/JSON medians. Reproduce with the recorded harness and production revisions:

```sh
python3 compat/bench/run.py /tmp/map-before --profile legacy-current --backend gogo \
  --current-revision 6fe392e3ce24d0cdc7dd0a8a1fbfd3d1d9e729b9 --count 5 --benchtime 200ms
python3 compat/bench/run.py /tmp/map-after --profile legacy-current --backend gogo \
  --current-revision 6816daa8787f6ab422f9d3084f03e83b7a62e842 --count 5 --benchtime 200ms
```

Validation includes concurrent omitted/explicit/overwritten closed-enum map defaults, default/purego/race runtime tests, frozen/historical compatibility, all Editions fast/auxiliary fixtures and both complete pinned 36.2 conformance profiles. The standard profile retains zero failures, four deliberately unsupported unstable-Edition cases and all 909 text cases passing.
