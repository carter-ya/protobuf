# Historical compiler CI

The maintained root declares Go1.15. The archived master-only workflow still
selected unsupported Go1.12 and attempted to regenerate today's Editions
metadata with protoc2.6/3.0. Its unversioned shadow-analyzer download also selected
a Go1.26-only tool. Those failures did not test the intended supported line.

The six-job matrix now uses supported Go1.15.15/Go1.23 and retains protoc2.6.1,
3.0.2 and 3.14.0. `run.sh` sends valid legacy schemas to each compiler and all
five maintained backends. Proto2 runs on every compiler; proto3 runs on the
compilers that implement it (3.0+). Each generated consumer verifies decoding,
marshal/size, binary and JSON round trips, clone isolation, oneofs and repeated
fields in default and purego builds. Backend defaults are preserved. Two valid
syntax profiles on four matrix jobs and one on two jobs total 100 consumer test
invocations. This does not advertise modern features on old compilers.

Every job separately uses pinned protoc36.2 for `full-suite.sh`: frozen-binding
and regenerated legacy AST/descriptor comparison, the complete root test suite,
the historical purego suite with the unchanged upstream #447/#448 exclusions,
and full `go vet`. The exact root x/tools pin supplies the shadow analyzer on
Go1.15; it is not fetched with an unversioned `go get`. The complete minimum-Go
matrix covers that analyzer. Tests run in temporary GOPATH-shaped copies;
the original checkout must remain clean. No regeneration, vet or purego failure
is converted into an allowed result.

The ordinary compatibility, conformance, published-version and candidate-root
workflows remain mandatory, including the modern Go1.23/Go1.27 profiles.
