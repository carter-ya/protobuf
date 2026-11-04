package proto

import (
	"reflect"
	"testing"
)

type zigzag32Overflow struct {
	Scalar   int32           `protobuf:"zigzag32,1,opt,name=scalar,proto3"`
	Optional *int32          `protobuf:"zigzag32,2,opt,name=optional"`
	Values   []int32         `protobuf:"zigzag32,3,rep,name=values"`
	Packed   []int32         `protobuf:"zigzag32,4,rep,packed,name=packed"`
	Mapping  map[int32]int32 `protobuf:"bytes,5,rep,name=mapping" protobuf_key:"zigzag32,1,opt,name=key" protobuf_val:"zigzag32,2,opt,name=value"`
}

func (m *zigzag32Overflow) Reset()       { *m = zigzag32Overflow{} }
func (*zigzag32Overflow) String() string { return "zigzag32Overflow" }
func (*zigzag32Overflow) ProtoMessage()  {}

func TestZigzag32TruncatesBeforeDecoding(t *testing.T) {
	// Protobuf truncates a sint32 wire varint to 32 bits before unzigzagging.
	// The extra high bits must not become the decoded value's sign bit.
	for _, tc := range []struct {
		wire uint64
		want int32
	}{
		{0, 0}, {1, -1}, {2, 1}, {0xffffffff, -2147483648},
		{0x100000000, 0}, {0x100000001, -1}, {0x100000002, 1},
		{0x1ffffffff, -2147483648}, {0x8000000000000000, 0},
		{0xffffffffffffffff, -2147483648},
	} {
		encoded := EncodeVarint(tc.wire)
		wire := []byte{8}
		wire = append(wire, encoded...)
		wire = append(wire, 16)
		wire = append(wire, encoded...)
		wire = append(wire, 24)
		wire = append(wire, encoded...)
		wire = append(wire, 34, byte(len(encoded)))
		wire = append(wire, encoded...)
		entry := []byte{8}
		entry = append(entry, encoded...)
		entry = append(entry, 16)
		entry = append(entry, encoded...)
		wire = append(wire, 42, byte(len(entry)))
		wire = append(wire, entry...)
		var got zigzag32Overflow
		if err := Unmarshal(wire, &got); err != nil {
			t.Fatalf("%x: %v", tc.wire, err)
		}
		want := zigzag32Overflow{Scalar: tc.want, Optional: Int32(tc.want), Values: []int32{tc.want}, Packed: []int32{tc.want}, Mapping: map[int32]int32{tc.want: tc.want}}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("wire=%x: got %+v, want %+v", tc.wire, got, want)
		}
		// The long-standing Buffer codec is an independent implementation.
		value, err := NewBuffer(encoded).DecodeZigzag32()
		if err != nil || int32(value) != tc.want {
			t.Fatalf("Buffer %x: %d %v", tc.wire, value, err)
		}
	}
}
