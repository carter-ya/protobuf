package gogoreflect_test

import (
	"bytes"
	"runtime"
	"testing"

	pb "github.com/gogo/protobuf/reflect/internal/testpb"
	v2 "google.golang.org/protobuf/proto"
)

func TestMergeDiscardClosedEnumLinearAllocation(t *testing.T) {
	n := new(pb.Closed)
	m := closedView(t, n)
	// Retain regular unknowns and unregistered native extension storage.
	m.SetUnknown(append(delimited(200, bytes.Repeat([]byte{1}, 32<<10)), varint(107, 9)...))
	previous := append([]byte(nil), m.GetUnknown()...)
	input := bytes.Repeat(append(varint(1, 5), varint(1, 99)...), 4000)
	options := v2.UnmarshalOptions{Merge: true, DiscardUnknown: true}
	if err := options.Unmarshal(nil, m.Interface()); err != nil {
		t.Fatal(err)
	}
	runtime.GC()
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	err := options.Unmarshal(input, m.Interface())
	runtime.ReadMemStats(&after)
	if err != nil || n.GetStatus() != pb.Status_GOOD || !bytes.Equal(m.GetUnknown(), previous) {
		t.Fatal("merge/discard changed known or existing unknown fields", err)
	}
	allocated := after.TotalAlloc - before.TotalAlloc
	t.Logf("allocated %d bytes for %d input bytes and %d existing unknown bytes", allocated, len(input), len(previous))
	if allocated > uint64(len(input)*256+len(previous)*16+(2<<20)) {
		t.Fatal("merge/discard repeatedly copied existing unknowns")
	}
	if err := options.Unmarshal([]byte{0x12, 1, 0x80}, m.Interface()); err == nil || !bytes.Equal(m.GetUnknown(), previous) {
		t.Fatal("malformed packed enum did not preserve prior unknowns", err)
	}
}

func TestOrdinaryUnknownDecodeLinearAllocation(t *testing.T) {
	n := new(pb.Node)
	m := view(t, n)
	input := bytes.Repeat(varint(200, 3), 8000)
	if err := v2.Unmarshal(nil, m.Interface()); err != nil {
		t.Fatal(err)
	}
	runtime.GC()
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	err := v2.Unmarshal(input, m.Interface())
	runtime.ReadMemStats(&after)
	if err != nil || !bytes.Equal(m.GetUnknown(), input) {
		t.Fatal("ordinary unknown fields were not retained", err)
	}
	allocated := after.TotalAlloc - before.TotalAlloc
	t.Logf("allocated %d bytes for %d unknown input bytes", allocated, len(input))
	if allocated > uint64(len(input)*256+(2<<20)) {
		t.Fatal("unknown decode repeatedly copied the accumulated buffer")
	}
}
