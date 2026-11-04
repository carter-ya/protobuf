package proto

import (
	"bytes"
	"reflect"
	"runtime"
	"testing"
	"time"
)

type optionalBadWireRecord struct {
	Time             *time.Time     `protobuf:"bytes,1,opt,name=time,proto3,proto3_optional,stdtime"`
	Duration         *time.Duration `protobuf:"bytes,2,opt,name=duration,proto3,proto3_optional,stdduration"`
	Value            *int32         `protobuf:"bytes,3,opt,name=value,proto3,proto3_optional,wktptr"`
	Count            int32          `protobuf:"varint,4,opt,name=count,proto3"`
	XXX_unrecognized []byte
}

func (m *optionalBadWireRecord) Reset()         { *m = optionalBadWireRecord{} }
func (m *optionalBadWireRecord) String() string { return CompactTextString(m) }
func (*optionalBadWireRecord) ProtoMessage()    {}

func TestOptionalStandardWrongWireRetainsInput(t *testing.T) {
	for _, bad := range [][]byte{{8, 1}, {21, 1, 2, 3, 4}, {25, 1, 2, 3, 4, 5, 6, 7, 8}, {11, 8, 1, 12}} {
		input := append(append([]byte{}, bad...), 32, 7)
		m := new(optionalBadWireRecord)
		if err := Unmarshal(input, m); err != nil {
			t.Fatalf("unknown wire occurrence %x: %v", bad, err)
		}
		if m.Count != 7 || m.Time != nil || m.Duration != nil || m.Value != nil || !bytes.Equal(m.XXX_unrecognized, bad) {
			t.Fatalf("wrong routing: %+v", m)
		}
		wire, err := Marshal(m)
		if err != nil {
			t.Fatal(err)
		}
		peer := new(optionalBadWireRecord)
		if err = Unmarshal(wire, peer); err != nil || !Equal(m, peer) {
			t.Fatalf("unknown roundtrip: %x %v", wire, err)
		}
	}
	if err := Unmarshal([]byte{10, 5, 8}, new(optionalBadWireRecord)); err == nil {
		t.Fatal("malformed length-delimited field accepted")
	}
}

type standardMapDefaultRecord struct {
	Values map[string]int32 `protobuf:"bytes,1,rep,name=values" protobuf_key:"bytes,1,opt,name=key" protobuf_val:"varint,2,opt,name=value,enum_closed=5:9,def=5"`
}

func (m *standardMapDefaultRecord) Reset()         { *m = standardMapDefaultRecord{} }
func (m *standardMapDefaultRecord) String() string { return CompactTextString(m) }
func (*standardMapDefaultRecord) ProtoMessage()    {}

func TestStandardTextMapEnumSchemaDefault(t *testing.T) {
	for _, tc := range []struct {
		text string
		want map[string]int32
	}{
		{`values { key:"a" }`, map[string]int32{"a": 5}},
		{`values {}`, map[string]int32{"": 5}},
		{`values: [{key:"a"}, {key:"b" value:9}]`, map[string]int32{"a": 5, "b": 9}},
	} {
		m := new(standardMapDefaultRecord)
		if err := UnmarshalTextStandard(tc.text, m); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(m.Values, tc.want) {
			t.Fatalf("%s: got %v want %v", tc.text, m.Values, tc.want)
		}
	}
	binary := new(standardMapDefaultRecord)
	if err := Unmarshal([]byte{10, 3, 10, 1, 'a'}, binary); err != nil || binary.Values["a"] != 5 {
		t.Fatalf("binary reference: %v %v", binary, err)
	}
	if err := UnmarshalTextStandard(`values { key:"a" value:0 }`, new(standardMapDefaultRecord)); err == nil {
		t.Fatal("undeclared closed enum accepted")
	}
}

type packedEnumTailRecord struct {
	Values           []int32 `protobuf:"varint,1,rep,packed,name=values,enum_closed=0:1"`
	Count            int32   `protobuf:"varint,2,opt,name=count,proto3"`
	XXX_unrecognized []byte
}

func (m *packedEnumTailRecord) Reset()         { *m = packedEnumTailRecord{} }
func (m *packedEnumTailRecord) String() string { return CompactTextString(m) }
func (*packedEnumTailRecord) ProtoMessage()    {}

func TestPackedClosedEnumsLinearAllocation(t *testing.T) {
	input := bytes.Repeat([]byte{10, 1, 2}, 16000)
	input = append(input, 16, 77)
	if err := Unmarshal([]byte{10, 1, 2}, new(packedEnumTailRecord)); err != nil {
		t.Fatal(err)
	}
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	m := new(packedEnumTailRecord)
	if err := Unmarshal(input, m); err != nil {
		t.Fatal(err)
	}
	runtime.ReadMemStats(&after)
	t.Logf("allocated %d bytes for %d input bytes", after.TotalAlloc-before.TotalAlloc, len(input))
	if m.Count != 77 || len(m.Values) != 0 || len(m.XXX_unrecognized) != 32000 {
		t.Fatalf("lost filtered values or tail: %+v", m)
	}
	if allocated := after.TotalAlloc - before.TotalAlloc; allocated > uint64(len(input))*128+(1<<20) {
		t.Fatalf("allocated %d bytes for %d input bytes; packed occurrence filtering must be linear", allocated, len(input))
	}
	mixed := new(packedEnumTailRecord)
	if err := Unmarshal([]byte{10, 3, 0, 2, 1, 16, 77}, mixed); err != nil || !reflect.DeepEqual(mixed.Values, []int32{0, 1}) || mixed.Count != 77 || !bytes.Equal(mixed.XXX_unrecognized, []byte{8, 2}) {
		t.Fatalf("mixed occurrence/tail: %+v %v", mixed, err)
	}
}

type omittedRequiredChild struct {
	X *int32 `protobuf:"varint,1,req,name=x"`
}

func (m *omittedRequiredChild) Reset()         { *m = omittedRequiredChild{} }
func (m *omittedRequiredChild) String() string { return CompactTextString(m) }
func (*omittedRequiredChild) ProtoMessage()    {}

type omittedRequiredMapRecord struct {
	Values map[string]*omittedRequiredChild `protobuf:"bytes,1,rep,name=values" protobuf_key:"bytes,1,opt,name=key" protobuf_val:"bytes,2,opt,name=value"`
}

func (m *omittedRequiredMapRecord) Reset()         { *m = omittedRequiredMapRecord{} }
func (m *omittedRequiredMapRecord) String() string { return CompactTextString(m) }
func (*omittedRequiredMapRecord) ProtoMessage()    {}

func TestStandardTextOmittedMapMessageRequired(t *testing.T) {
	for _, input := range []string{`values { key:"a" }`, `values {}`, `values { key:"a" value:{} }`} {
		err := UnmarshalTextStandard(input, new(omittedRequiredMapRecord))
		if _, ok := err.(*RequiredNotSetError); !ok {
			t.Fatalf("%s: want RequiredNotSetError, got %v", input, err)
		}
	}
	m := new(omittedRequiredMapRecord)
	if err := UnmarshalTextStandard(`values { key:"a" value:{x:7} }`, m); err != nil || m.Values["a"] == nil || *m.Values["a"].X != 7 {
		t.Fatalf("valid message map: %v %v", m, err)
	}
}

type packedExtensionTailRecord struct {
	XXX_InternalExtensions
	Count            int32 `protobuf:"varint,1,opt,name=count,proto3"`
	XXX_unrecognized []byte
}

func (m *packedExtensionTailRecord) Reset()         { *m = packedExtensionTailRecord{} }
func (m *packedExtensionTailRecord) String() string { return "packed extension tail" }
func (*packedExtensionTailRecord) ProtoMessage()    {}
func (*packedExtensionTailRecord) ExtensionRangeArray() []ExtensionRange {
	return []ExtensionRange{{100, 199}}
}

var packedTailExtension = &ExtensionDesc{ExtendedType: (*packedExtensionTailRecord)(nil), ExtensionType: []int32(nil), Field: 150, Name: "review.packed_tail", Tag: "varint,150,rep,packed,name=packed_tail,enum_closed=0:1"}

func init() { RegisterExtension(packedTailExtension) }
func TestPackedClosedEnumExtensionRetainsTail(t *testing.T) {
	input := append(extensionWire(150, []byte{0, 2, 1}), 8, 77)
	m := new(packedExtensionTailRecord)
	if err := Unmarshal(input, m); err != nil {
		t.Fatal(err)
	}
	values, err := GetExtension(m, packedTailExtension)
	if err != nil || m.Count != 77 || !reflect.DeepEqual(values, []int32{0, 1}) || !bytes.Equal(m.XXX_unrecognized, []byte{0xb0, 9, 2}) {
		t.Fatalf("filtered extension/tail: %v %v %v", m, values, err)
	}
	wire, err := Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	peer := new(packedExtensionTailRecord)
	if err = Unmarshal(wire, peer); err != nil || !Equal(m, peer) {
		t.Fatalf("extension roundtrip: %v", err)
	}
}

type omittedOptionalMapRecord struct {
	Values map[string]*extensionValidationPayload `protobuf:"bytes,1,rep,name=values" protobuf_key:"bytes,1,opt,name=key" protobuf_val:"bytes,2,opt,name=value"`
}

func (m *omittedOptionalMapRecord) Reset()         { *m = omittedOptionalMapRecord{} }
func (m *omittedOptionalMapRecord) String() string { return CompactTextString(m) }
func (*omittedOptionalMapRecord) ProtoMessage()    {}
func TestStandardTextOmittedOptionalMapRepresentation(t *testing.T) {
	m := new(omittedOptionalMapRecord)
	if err := UnmarshalTextStandard(`values {key:"a"}`, m); err != nil || len(m.Values) != 1 || m.Values["a"] != nil {
		t.Fatalf("nil default representation changed: %+v %v", m, err)
	}
}
