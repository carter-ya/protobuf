package proto

import (
	"bytes"
	"testing"
)

type mapWireMessage struct {
	Values           map[int32]string `protobuf:"bytes,1,rep,name=values,proto3" protobuf_key:"varint,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
	Tail             int32            `protobuf:"varint,2,opt,name=tail,proto3"`
	XXX_unrecognized []byte
}

func (m *mapWireMessage) Reset()         { *m = mapWireMessage{} }
func (m *mapWireMessage) String() string { return CompactTextString(m) }
func (*mapWireMessage) ProtoMessage()    {}
func TestMapMismatchedWireTypesAreUnknown(t *testing.T) {
	for _, unknown := range [][]byte{{8, 1}, {9, 1, 2, 3, 4, 5, 6, 7, 8}, {11, 16, 1, 12}, {13, 1, 2, 3, 4}} {
		input := append(append([]byte{}, unknown...), 16, 7)
		var got mapWireMessage
		if err := Unmarshal(input, &got); err != nil {
			t.Fatalf("%x: %v", input, err)
		}
		if got.Values != nil || got.Tail != 7 || !bytes.Equal(got.XXX_unrecognized, unknown) {
			t.Fatalf("map/continuation/unknown changed: %+v", got)
		}
		output, err := Marshal(&got)
		if err != nil {
			t.Fatal(err)
		}
		var back mapWireMessage
		if err := Unmarshal(output, &back); err != nil || !Equal(&got, &back) {
			t.Fatalf("roundtrip: %v", err)
		}
	}
	for _, input := range [][]byte{{8}, {9, 1}, {11, 16, 1}, {13, 1}, {10, 1, 0}, {10, 2, 8}} {
		if err := Unmarshal(input, new(mapWireMessage)); err == nil {
			t.Fatalf("accepted malformed field %x", input)
		}
	}
	var got mapWireMessage
	if err := Unmarshal([]byte{10, 5, 8, 3, 18, 1, 'x'}, &got); err != nil || got.Values[3] != "x" {
		t.Fatalf("ordinary map changed: %v", err)
	}
}
