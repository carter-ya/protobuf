package legacy_test

import (
	"bytes"
	"testing"

	"example.com/gogo-legacy/fast"
	"example.com/gogo-legacy/reflection"
	"github.com/gogo/protobuf/jsonpb"
	"github.com/gogo/protobuf/proto"
)

// These types are deliberately frozen output of the unmodified v1.3.2 generator.
func TestFrozenGeneratedCode(t *testing.T) {
	for _, m := range []proto.Message{
		&reflection.Probe{Value: 42, Label: "old", Data: []byte("data"), Numbers: []int32{0, 1, -1}, Counts: map[string]int32{"a": 1}, Choice: &reflection.Probe_Name{Name: "name"}},
		&fast.Probe{Value: 42, Label: "old", Data: []byte("data"), Numbers: []int32{0, 1, -1}, Choice: &fast.Probe_Id{Id: 7}},
	} {
		b, err := proto.Marshal(m)
		if err != nil {
			t.Fatal(err)
		}
		if proto.Size(m) != len(b) {
			t.Fatalf("%T: size mismatch", m)
		}
		clone := proto.Clone(m)
		clone.Reset()
		if err := proto.Unmarshal(b, clone); err != nil {
			t.Fatal(err)
		}
		if !proto.Equal(m, clone) {
			t.Fatalf("%T: round trip lost data", m)
		}
		j, err := (&jsonpb.Marshaler{}).MarshalToString(m)
		if err != nil {
			t.Fatal(err)
		}
		clone.Reset()
		if err := jsonpb.UnmarshalString(j, clone); err != nil {
			t.Fatal(err)
		}
		if !proto.Equal(m, clone) {
			t.Fatalf("%T: JSON round trip lost data", m)
		}
	}
}

func TestFrozenWireAndUnknownFields(t *testing.T) {
	m := &reflection.Probe{Value: 42}
	b, err := proto.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(b, []byte{8, 42}) {
		t.Fatalf("wire changed: %x", b)
	}
	// Unknown field 99 must survive in the reflection variant.
	b = append(b, 0x98, 0x06, 0x07)
	if err := proto.Unmarshal(b, m); err != nil {
		t.Fatal(err)
	}
	out, err := proto.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(out, b) {
		t.Fatalf("unknown field lost: %x", out)
	}
	proto.Merge(m, &reflection.Probe{Label: "merged"})
	if m.Value != 42 || m.Label != "merged" {
		t.Fatal("merge semantics changed")
	}
	if proto.MessageName(m) != "compat.legacy.Probe" {
		t.Fatal("registration name changed")
	}
}
