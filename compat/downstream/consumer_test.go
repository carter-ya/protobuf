package downstream

import (
	"context"
	errors "github.com/cockroachdb/errors"
	"github.com/cockroachdb/errors/extgrpc"
	"github.com/gogo/protobuf/jsonpb"
	"github.com/gogo/protobuf/proto"
	"go.etcd.io/etcd/raft/v3/raftpb"
	"google.golang.org/grpc/codes"
	"reflect"
	"testing"
)

func TestFrozenRaftFastTypes(t *testing.T) {
	for _, m := range []proto.Message{
		&raftpb.Entry{Term: 3, Index: 4, Type: raftpb.EntryConfChangeV2, Data: []byte("payload")},
		&raftpb.Snapshot{Data: []byte("snapshot"), Metadata: raftpb.SnapshotMetadata{Index: 4, Term: 3, ConfState: raftpb.ConfState{Voters: []uint64{1, 2}}}},
		&raftpb.ConfChangeV2{Changes: []raftpb.ConfChangeSingle{{Type: raftpb.ConfChangeAddNode, NodeID: 3}}},
	} {
		b, err := proto.Marshal(m)
		if err != nil || proto.Size(m) != len(b) {
			t.Fatalf("fast marshal: %v", err)
		}
		back := reflect.New(reflect.TypeOf(m).Elem()).Interface().(proto.Message)
		if err := proto.Unmarshal(b, back); err != nil || !proto.Equal(m, back) {
			t.Fatalf("fast roundtrip: %v", err)
		}
		if !proto.Equal(m, proto.Clone(m)) {
			t.Fatal("fast type clone changed")
		}
		j, err := (&jsonpb.Marshaler{EnumsAsInts: true}).MarshalToString(m)
		if err != nil {
			t.Fatal(err)
		}
		back.Reset()
		if err := jsonpb.UnmarshalString(j, back); err != nil || !proto.Equal(m, back) {
			t.Fatalf("raft JSON: %s %v", j, err)
		}
	}
}
func TestFrozenErrorsGrpcJSONAndAny(t *testing.T) {
	ctx := context.Background()
	original := errors.WithDetail(extgrpc.WrapWithGrpcCode(errors.New("missing"), codes.NotFound), "detail")
	encoded := errors.EncodeError(ctx, original)
	b, err := proto.Marshal(&encoded)
	if err != nil {
		t.Fatal(err)
	}
	var restored errors.EncodedError
	if err := proto.Unmarshal(b, &restored); err != nil || !proto.Equal(&encoded, &restored) {
		t.Fatalf("error protobuf: %v", err)
	}
	j, err := (&jsonpb.Marshaler{}).MarshalToString(&encoded)
	if err != nil {
		t.Fatal(err)
	}
	restored.Reset()
	if err := jsonpb.UnmarshalString(j, &restored); err != nil {
		t.Fatalf("error JSON/Any: %s %v", j, err)
	}
	decoded := errors.DecodeError(ctx, restored)
	if extgrpc.GetGrpcCode(decoded) != codes.NotFound || !reflect.DeepEqual(errors.GetAllDetails(decoded), []string{"detail"}) {
		t.Fatalf("error details or grpc code changed: %v", decoded)
	}
}
