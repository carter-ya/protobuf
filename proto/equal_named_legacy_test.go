package proto_test

import (
	"github.com/gogo/protobuf/proto"
	"testing"
)

type legacyName string
type legacyBytes []byte
type legacyByte uint8
type legacyNamedByteSlice []legacyByte
type legacyNamedValues struct {
	Name   legacyName            `protobuf:"bytes,1,opt,name=name,casttype=legacyName"`
	Data   legacyBytes           `protobuf:"bytes,2,opt,name=data,casttype=legacyBytes"`
	Names  []legacyName          `protobuf:"bytes,3,rep,name=names,casttype=legacyName"`
	Values map[string]legacyName `protobuf:"bytes,4,rep,name=values,castvalue=legacyName" protobuf_key:"bytes,1,opt,name=key" protobuf_val:"bytes,2,opt,name=value"`
	Other  legacyNamedByteSlice  `protobuf:"bytes,5,opt,name=other,casttype=legacyNamedByteSlice"`
}

func (m *legacyNamedValues) Reset()         { *m = legacyNamedValues{} }
func (m *legacyNamedValues) String() string { return proto.CompactTextString(m) }
func (*legacyNamedValues) ProtoMessage()    {}
func TestLegacyNamedScalarEquality(t *testing.T) {
	newValue := func() *legacyNamedValues {
		return &legacyNamedValues{Name: "uid", Data: legacyBytes{1, 2}, Names: []legacyName{"name"}, Values: map[string]legacyName{"key": "value"}, Other: legacyNamedByteSlice{3, 4}}
	}
	a, b := newValue(), newValue()
	if !proto.Equal(a, b) {
		t.Fatal("equal named values differ")
	}
	mutations := []func(*legacyNamedValues){func(v *legacyNamedValues) { v.Name = "other" }, func(v *legacyNamedValues) { v.Data[0]++ }, func(v *legacyNamedValues) { v.Names[0] = "other" }, func(v *legacyNamedValues) { v.Values["key"] = "other" }, func(v *legacyNamedValues) { v.Other[0]++ }}
	for _, mutate := range mutations {
		b := newValue()
		mutate(b)
		if proto.Equal(a, b) {
			t.Fatal("unequal named values compare equal")
		}
	}
	if proto.Equal(&legacyNamedValues{}, &legacyNamedValues{Data: legacyBytes{}}) {
		t.Fatal("proto2 named bytes lost nil/empty distinction")
	}
}

var legacyNamedMergeInfo proto.InternalMessageInfo

func (m *legacyNamedValues) XXX_Merge(src proto.Message) { legacyNamedMergeInfo.Merge(m, src) }

func TestLegacyNamedScalarTableMerge(t *testing.T) {
	original := &legacyNamedValues{Name: "uid", Data: legacyBytes{1, 2}, Names: []legacyName{"name"}, Values: map[string]legacyName{"key": "value"}, Other: legacyNamedByteSlice{3, 4}}
	cloned := proto.Clone(original).(*legacyNamedValues)
	if !proto.Equal(original, cloned) {
		t.Fatal("table clone changed named fields")
	}
	cloned.Data[0]++
	cloned.Names[0] = "other"
	cloned.Values["key"] = "other"
	cloned.Other[0]++
	if original.Data[0] != 1 || original.Names[0] != "name" || original.Values["key"] != "value" || original.Other[0] != 3 {
		t.Fatal("clone aliases mutable named storage")
	}
	dst := &legacyNamedValues{Name: "keep", Data: legacyBytes{9}}
	proto.Merge(dst, &legacyNamedValues{Data: legacyBytes{}})
	if dst.Name != "keep" || dst.Data == nil || len(dst.Data) != 0 {
		t.Fatal("legacy scalar zero/bytes presence changed")
	}
}
