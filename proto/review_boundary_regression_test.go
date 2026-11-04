package proto

import (
	"bytes"
	"reflect"
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
