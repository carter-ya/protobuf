package proto

import (
	"bytes"
	"testing"
)

type wireValidationGroup struct {
	Value int32 `protobuf:"varint,3,opt,name=value,proto3"`
}

func (m *wireValidationGroup) Reset()       { *m = wireValidationGroup{} }
func (*wireValidationGroup) String() string { return "wireValidationGroup" }
func (*wireValidationGroup) ProtoMessage()  {}

type wireValidationMessage struct {
	Group            *wireValidationGroup   `protobuf:"group,1,opt,name=group"`
	Groups           []*wireValidationGroup `protobuf:"group,2,rep,name=groups"`
	Mapping          map[int32]int32        `protobuf:"bytes,3,rep,name=mapping" protobuf_key:"varint,1,opt,name=key" protobuf_val:"varint,2,opt,name=value"`
	XXX_unrecognized []byte
}

func (m *wireValidationMessage) Reset()       { *m = wireValidationMessage{} }
func (*wireValidationMessage) String() string { return "wireValidationMessage" }
func (*wireValidationMessage) ProtoMessage()  {}

func TestReflectionWireTagAndGroupValidation(t *testing.T) {
	for _, tc := range []struct {
		name string
		wire []byte
	}{
		{"zero field", []byte{0, 0}},
		{"field beyond limit", append(EncodeVarint(0x20000000<<3), 0)},
		{"field far beyond limit", append(EncodeVarint(0x100000001<<3), 0)},
		{"overlong field header", []byte{0x88, 0x80, 0x80, 0x80, 0x80, 0, 0}},
		{"known group mismatch", []byte{11, 20}},
		{"repeated group mismatch", []byte{19, 12}},
		{"unknown group mismatch", []byte{43, 52}},
		{"unknown nested group mismatch", []byte{43, 51, 60, 44}},
		{"known nested group mismatch", []byte{11, 51, 60, 12}},
		{"zero inside group", []byte{43, 0, 0, 44}},
		{"bad tag inside group", append(append([]byte{43}, EncodeVarint(0x20000000<<3)...), 0, 44)},
		{"overlong tag inside group", []byte{43, 0x88, 0x80, 0x80, 0x80, 0x80, 0, 0, 44}},
		{"mismatched group in map", []byte{26, 2, 43, 52}},
		{"zero inside map", []byte{26, 2, 0, 0}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := Unmarshal(tc.wire, new(wireValidationMessage)); err == nil {
				t.Fatalf("accepted malformed wire %x", tc.wire)
			}
		})
	}
	// Highest legal tag and matching nested unknown groups retain exact bytes.
	unknown := append(EncodeVarint(0x1fffffff<<3), 7)
	unknown = append(unknown, 43, 51, 8, 1, 52, 44)
	wire := append([]byte{11, 24, 7, 12, 19, 24, 8, 20}, unknown...)
	var value wireValidationMessage
	if err := Unmarshal(wire, &value); err != nil {
		t.Fatal(err)
	}
	if value.Group == nil || value.Group.Value != 7 || len(value.Groups) != 1 || value.Groups[0].Value != 8 || !bytes.Equal(value.XXX_unrecognized, unknown) {
		t.Fatalf("valid data changed: %+v", value)
	}
	// Buffer.DecodeGroup has no outer field number; nested group tags still
	// must match, and the closing tag must be a valid field key.
	if err := NewBuffer([]byte{51, 60, 44}).DecodeGroup(new(wireValidationGroup)); err == nil {
		t.Fatal("Buffer accepted mismatched nested group")
	}
}

func TestReflectionGroupNestingLimit(t *testing.T) {
	for _, depth := range []int{16, 10000, 10001} {
		wire := append(bytes.Repeat([]byte{43}, depth), bytes.Repeat([]byte{44}, depth)...)
		err := Unmarshal(wire, new(wireValidationMessage))
		if depth <= 10000 && err != nil || depth > 10000 && err == nil {
			t.Fatalf("group depth %d: %v", depth, err)
		}
	}
}
