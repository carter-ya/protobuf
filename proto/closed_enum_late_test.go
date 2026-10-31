package proto_test

import (
	"github.com/gogo/protobuf/proto"
	"testing"
)

type lateClosedEnumHost struct {
	proto.XXX_InternalExtensions
	XXX_unrecognized []byte
}

func (m *lateClosedEnumHost) Reset()         { *m = lateClosedEnumHost{} }
func (m *lateClosedEnumHost) String() string { return "late closed enum host" }
func (*lateClosedEnumHost) ProtoMessage()    {}
func (*lateClosedEnumHost) ExtensionRangeArray() []proto.ExtensionRange {
	return []proto.ExtensionRange{{Start: 100, End: 200}}
}
func TestLateClosedEnumRegistration(t *testing.T) {
	var value lateClosedEnumHost
	if err := proto.Unmarshal(nil, &value); err != nil {
		t.Fatal(err)
	}
	extension := &proto.ExtensionDesc{ExtendedType: (*lateClosedEnumHost)(nil), ExtensionType: (*int32)(nil), Field: 150, Name: "late_closed_enum", Tag: "varint,150,opt,name=late_closed_enum,enum_closed=5:6,def=5"}
	if proto.RegisteredExtensions(&value)[150] == nil {
		proto.RegisterExtension(extension)
	}
	wire := []byte{0xb0, 0x09, 99}
	if err := proto.Unmarshal(wire, &value); err != nil {
		t.Fatal(err)
	}
	if proto.HasExtension(&value, extension) {
		t.Fatal("late unknown closed enum became a known extension")
	}
	encoded, err := proto.Marshal(&value)
	if err != nil {
		t.Fatal(err)
	}
	if string(encoded) != string(wire) {
		t.Fatal("late unknown closed enum bytes were lost")
	}
}

type lateClosedEnumBytesHost struct {
	XXX_extensions   []byte
	XXX_unrecognized []byte
}

func (m *lateClosedEnumBytesHost) Reset()                 { *m = lateClosedEnumBytesHost{} }
func (m *lateClosedEnumBytesHost) String() string         { return "late byte closed enum host" }
func (*lateClosedEnumBytesHost) ProtoMessage()            {}
func (m *lateClosedEnumBytesHost) GetExtensions() *[]byte { return &m.XXX_extensions }
func (*lateClosedEnumBytesHost) ExtensionRangeArray() []proto.ExtensionRange {
	return []proto.ExtensionRange{{Start: 100, End: 200}}
}
func TestLateClosedEnumBytesRegistration(t *testing.T) {
	var value lateClosedEnumBytesHost
	if err := proto.Unmarshal(nil, &value); err != nil {
		t.Fatal(err)
	}
	extension := &proto.ExtensionDesc{ExtendedType: (*lateClosedEnumBytesHost)(nil), ExtensionType: (*int32)(nil), Field: 150, Name: "late_byte_closed_enum", Tag: "varint,150,opt,name=late_byte_closed_enum,enum_closed=5:6,def=5"}
	if proto.RegisteredExtensions(&value)[150] == nil {
		proto.RegisterExtension(extension)
	}
	wire := []byte{0xb0, 0x09, 99}
	if err := proto.Unmarshal(wire, &value); err != nil {
		t.Fatal(err)
	}
	if proto.HasExtension(&value, extension) {
		t.Fatal("late unknown closed enum became a known byte extension")
	}
	encoded, err := proto.Marshal(&value)
	if err != nil || string(encoded) != string(wire) {
		t.Fatal("late unknown closed enum bytes were lost", encoded, err)
	}
}
