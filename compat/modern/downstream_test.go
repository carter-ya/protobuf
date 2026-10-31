package modern

import (
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"strings"
	"testing"

	cockroach "github.com/cockroachdb/errors"
	"github.com/cockroachdb/errors/errorspb"
	gogo "github.com/gogo/protobuf/proto"
	"go.etcd.io/etcd/raft/v3/raftpb"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
)

// Use the actual downstream's frozen descriptor, not a recompiled substitute.
// Older gogo descriptors spell their gogo import with the historical full path.
func frozenRegistry(messages ...interface{ Descriptor() ([]byte, []int) }) (*Registry, error) {
	set := new(descriptorpb.FileDescriptorSet)
	seen := map[string]bool{}
	var add func([]byte, string) error
	add = func(raw []byte, path string) error {
		if len(raw) == 0 {
			return fmt.Errorf("missing frozen dependency %s", path)
		}
		z, err := gzip.NewReader(bytes.NewReader(raw))
		if err != nil {
			return err
		}
		data, err := io.ReadAll(z)
		z.Close()
		if err != nil {
			return err
		}
		file := new(descriptorpb.FileDescriptorProto)
		if err := proto.Unmarshal(data, file); err != nil {
			return err
		}
		if path != "" {
			file.Name = proto.String(path)
		}
		if seen[file.GetName()] {
			return nil
		}
		seen[file.GetName()] = true
		set.File = append(set.File, file)
		for _, dependency := range file.Dependency {
			raw := gogo.FileDescriptor(dependency)
			if len(raw) == 0 {
				raw = gogo.FileDescriptor(strings.TrimPrefix(dependency, "github.com/gogo/protobuf/"))
			}
			if len(raw) == 0 && strings.HasSuffix(dependency, "/gogo.proto") {
				raw = gogo.FileDescriptor("gogo.proto")
			}
			if len(raw) == 0 && dependency == "google/protobuf/descriptor.proto" {
				raw = gogo.FileDescriptor("descriptor.proto")
			}
			if err := add(raw, dependency); err != nil {
				return err
			}
		}
		return nil
	}
	for _, m := range messages {
		raw, _ := m.Descriptor()
		if err := add(raw, ""); err != nil {
			return nil, err
		}
	}
	wire, err := proto.Marshal(set)
	if err != nil {
		return nil, err
	}
	return NewRegistry(wire)
}

func TestFrozenDownstreamSnapshots(t *testing.T) {
	encoded := cockroach.EncodeError(context.Background(), cockroach.WithDetail(cockroach.New("missing"), "detail"))
	for _, original := range []gogo.Message{
		&raftpb.Entry{Term: 3, Index: 4, Type: raftpb.EntryConfChangeV2, Data: []byte("payload")},
		&encoded,
	} {
		r, err := frozenRegistry(original.(interface{ Descriptor() ([]byte, []int) }), new(errorspb.StringPayload))
		if err != nil {
			t.Fatal(err)
		}
		before := gogo.Clone(original)
		m, err := r.Snapshot(original)
		if err != nil {
			t.Fatal(err)
		}
		text, err := (protojson.MarshalOptions{Resolver: r.Types}).Marshal(m)
		if err != nil {
			t.Fatal(err)
		}
		m.Reset()
		if err := (protojson.UnmarshalOptions{Resolver: r.Types}).Unmarshal(text, m); err != nil {
			t.Fatalf("%s: %v", text, err)
		}
		if err := r.Apply(original, m); err != nil || !gogo.Equal(before, original) {
			t.Fatalf("frozen downstream changed: %v", err)
		}
	}
	decoded := cockroach.DecodeError(context.Background(), encoded)
	if decoded.Error() != "missing" || strings.Join(cockroach.GetAllDetails(decoded), ",") != "detail" {
		t.Fatal("downstream error semantics changed")
	}
}
