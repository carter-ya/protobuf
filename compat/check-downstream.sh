#!/bin/sh
set -eu
repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT HUP INT TERM
cd "$repo/compat/downstream"
# Fixed original generated files: runtime-only upgrade, no application edits.
go test ./... github.com/cockroachdb/errors/errbase github.com/cockroachdb/errors/extgrpc go.etcd.io/etcd/raft/v3/...
raft=$(go list -m -f '{{.Dir}}' go.etcd.io/etcd/raft/v3)
cp -R "$raft" "$tmp/raft"
chmod -R u+w "$tmp/raft"
mkdir -p "$tmp/original" "$tmp/reference" "$tmp/current"
cd "$repo"
git archive v1.3.2 | tar -xf - -C "$tmp/original"
(cd "$tmp/original" && go build -o "$tmp/original-gofast" ./protoc-gen-gofast)
go build -o "$tmp/current-gofast" ./protoc-gen-gofast
# Etcd's frozen types use the golang runtime: preserve the gofast entrypoint.
protoc -I "$raft/raftpb" -I . -I protobuf --plugin=protoc-gen-test="$tmp/original-gofast" --test_out=paths=source_relative:"$tmp/reference" raft.proto
protoc -I "$raft/raftpb" -I . -I protobuf --plugin=protoc-gen-test="$tmp/current-gofast" --test_out=paths=source_relative:"$tmp/current" raft.proto
go run ./compat/compare-generated -- "$tmp/reference/raft.pb.go" "$tmp/current/raft.pb.go"
cp "$tmp/current/raft.pb.go" "$tmp/raft/raftpb/raft.pb.go"
cp compat/downstream/go.mod "$tmp/regenerated.mod"
cp compat/downstream/go.sum "$tmp/regenerated.sum"
cd "$repo/compat/downstream"
go mod edit -modfile="$tmp/regenerated.mod" -replace=github.com/gogo/protobuf="$repo" -replace=go.etcd.io/etcd/raft/v3="$tmp/raft"
go test -modfile="$tmp/regenerated.mod" ./... go.etcd.io/etcd/raft/v3/...
