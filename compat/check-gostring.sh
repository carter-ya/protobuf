#!/bin/sh
set -eu
repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
cd "$repo"
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT HUP INT TERM
go build -o "$tmp/gogo" ./protoc-gen-gogo
cat > "$tmp/fixture.proto" <<'EOF'
syntax = "proto3";
package fixture;
option go_package = "example.com/gostring-fixture;fixture";
import "gogoproto/gogo.proto";
import "google/protobuf/timestamp.proto";
option (gogoproto.gostring_all) = true;
message Sample { google.protobuf.Timestamp value = 1 [(gogoproto.stdtime) = true]; }
EOF
protoc -I "$tmp" -I . -I protobuf --plugin=protoc-gen-test="$tmp/gogo" --test_out=paths=source_relative,Mgoogle/protobuf/timestamp.proto=github.com/gogo/protobuf/types:"$tmp" fixture.proto
cat > "$tmp/go.mod" <<EOF
module example.com/gostring-fixture
go 1.15
require github.com/gogo/protobuf v1.3.2
replace github.com/gogo/protobuf => $repo
EOF
cp compat/legacy/go.sum "$tmp/go.sum"
cat > "$tmp/fixture_test.go" <<'EOF'
package fixture
import("go/parser";"testing")
func TestGeneratedNilTimestamp(t *testing.T){if _,err:=parser.ParseExpr((&Sample{}).GoString());err!=nil{t.Fatal(err)}}
EOF
(cd "$tmp" && go test ./...)
