package standard

import (
	"bytes"
	"compress/gzip"
	"strings"
	"testing"

	"github.com/gogo/protobuf/proto"
	descriptor "github.com/gogo/protobuf/protoc-gen-gogo/descriptor"
)

type groupLeaf struct {
	Value int32 `protobuf:"varint,1,opt,name=value,proto3"`
}

func (m *groupLeaf) Reset()         { *m = groupLeaf{} }
func (m *groupLeaf) String() string { return proto.CompactTextString(m) }
func (*groupLeaf) ProtoMessage()    {}

type groupProbe struct {
	Multi     *groupLeaf `protobuf:"group,1,opt,name=Multiwordgroupfield"`
	Delimited *groupLeaf `protobuf:"group,2,opt,name=groupliketype,proto3,explicit"`
	Other     *groupLeaf `protobuf:"group,3,opt,name=other,proto3,explicit"`
	Foreign   *groupLeaf `protobuf:"group,4,opt,name=foreign,proto3,explicit"`
	Plain     *groupLeaf `protobuf:"bytes,5,opt,name=plain,proto3"`
	Upper     string     `protobuf:"bytes,6,opt,name=Data,proto3"`
	Lower     *groupLeaf `protobuf:"group,7,opt,name=Data"`
}

func (m *groupProbe) Reset()                    { *m = groupProbe{} }
func (m *groupProbe) String() string            { return proto.CompactTextString(m) }
func (*groupProbe) ProtoMessage()               {}
func (*groupProbe) Descriptor() ([]byte, []int) { return groupDescriptor, []int{0} }

var groupDescriptor = func() []byte {
	field := func(name string, number int32, kind descriptor.FieldDescriptorProto_Type, typ string) *descriptor.FieldDescriptorProto {
		return &descriptor.FieldDescriptorProto{Name: proto.String(name), Number: proto.Int32(number), Type: &kind, TypeName: proto.String(typ)}
	}
	file := &descriptor.FileDescriptorProto{Package: proto.String("text"), MessageType: []*descriptor.DescriptorProto{{
		Name: proto.String("Groups"),
		Field: []*descriptor.FieldDescriptorProto{
			field("multiwordgroupfield", 1, descriptor.FieldDescriptorProto_TYPE_GROUP, ".text.Groups.MultiWordGroupField"),
			field("groupliketype", 2, descriptor.FieldDescriptorProto_TYPE_MESSAGE, ".text.Groups.GroupLikeType"),
			field("other", 3, descriptor.FieldDescriptorProto_TYPE_MESSAGE, ".text.Groups.GroupLikeType"),
			field("foreign", 4, descriptor.FieldDescriptorProto_TYPE_MESSAGE, ".text.Foreign"),
			field("plain", 5, descriptor.FieldDescriptorProto_TYPE_MESSAGE, ".text.Groups.Plain"),
			field("Data", 6, descriptor.FieldDescriptorProto_TYPE_STRING, ""),
			field("data", 7, descriptor.FieldDescriptorProto_TYPE_GROUP, ".text.Groups.Data"),
		},
		NestedType: []*descriptor.DescriptorProto{{Name: proto.String("MultiWordGroupField")}, {Name: proto.String("GroupLikeType")}, {Name: proto.String("Plain")}, {Name: proto.String("Data")}},
	}, {Name: proto.String("Foreign")}}}
	wire, err := proto.Marshal(file)
	if err != nil {
		panic(err)
	}
	var out bytes.Buffer
	writer := gzip.NewWriter(&out)
	if _, err := writer.Write(wire); err != nil {
		panic(err)
	}
	if err := writer.Close(); err != nil {
		panic(err)
	}
	return out.Bytes()
}()

func TestStandardGroupNames(t *testing.T) {
	for _, name := range []string{"MultiWordGroupField", "multiwordgroupfield", "GroupLikeType", "groupliketype", "other", "foreign", "plain", "data"} {
		var value groupProbe
		if err := UnmarshalString(name+` { value: 1 }`, &value); err != nil {
			t.Errorf("%s: %v", name, err)
		}
	}
	for _, name := range []string{"Multiwordgroupfield", "MULTIWORDGROUPFIELD", "GROUPLIKETYPE", "Foreign", "Plain"} {
		if err := UnmarshalString(name+` { value: 1 }`, new(groupProbe)); err == nil {
			t.Errorf("accepted %s", name)
		}
	}
	if err := UnmarshalString(`MultiWordGroupField {} multiwordgroupfield {}`, new(groupProbe)); err == nil {
		t.Fatal("group alias bypassed singular duplication check")
	}
	var value groupProbe
	if err := UnmarshalString(`Data: "upper" data { value: 7 }`, &value); err != nil || value.Upper != "upper" || value.Lower == nil || value.Lower.Value != 7 {
		t.Fatalf("field name precedence: %#v %v", value, err)
	}
	text, err := (&Marshaler{}).MarshalToString(&groupProbe{Multi: &groupLeaf{Value: 1}, Delimited: &groupLeaf{Value: 2}, Other: &groupLeaf{Value: 3}})
	if err != nil || !strings.Contains(text, "MultiWordGroupField") || !strings.Contains(text, "GroupLikeType") || !strings.Contains(text, "other") {
		t.Fatalf("group output: %q %v", text, err)
	}
	var back groupProbe
	if err := UnmarshalString(text, &back); err != nil || back.Multi == nil || back.Multi.Value != 1 || back.Delimited == nil || back.Delimited.Value != 2 {
		t.Fatalf("group roundtrip: %v", err)
	}
	if err := proto.UnmarshalText(`Multiwordgroupfield { value: 1 }`, new(groupProbe)); err != nil {
		t.Fatalf("legacy group spelling changed: %v", err)
	}
}

type groupOneofProbe struct {
	Choice groupChoice `protobuf_oneof:"choice"`
}
type groupChoice interface{ isGroupChoice() }
type groupVariant struct {
	Group *groupLeaf `protobuf:"group,1,opt,name=groupliketype,proto3,oneof,explicit"`
}

func (*groupVariant) isGroupChoice()                      {}
func (m *groupOneofProbe) Reset()                         { *m = groupOneofProbe{} }
func (m *groupOneofProbe) String() string                 { return proto.CompactTextString(m) }
func (*groupOneofProbe) ProtoMessage()                    {}
func (*groupOneofProbe) XXX_OneofWrappers() []interface{} { return []interface{}{(*groupVariant)(nil)} }
func (*groupOneofProbe) Descriptor() ([]byte, []int)      { return oneofGroupDescriptor, []int{0} }

var oneofGroupDescriptor = func() []byte {
	file := &descriptor.FileDescriptorProto{Package: proto.String("text"), MessageType: []*descriptor.DescriptorProto{{
		Name:       proto.String("OneofGroup"),
		Field:      []*descriptor.FieldDescriptorProto{{Name: proto.String("groupliketype"), Number: proto.Int32(1), Type: descriptor.FieldDescriptorProto_TYPE_MESSAGE.Enum(), TypeName: proto.String(".text.OneofGroup.GroupLikeType"), OneofIndex: proto.Int32(0)}},
		NestedType: []*descriptor.DescriptorProto{{Name: proto.String("GroupLikeType")}},
		OneofDecl:  []*descriptor.OneofDescriptorProto{{Name: proto.String("choice")}},
	}}}
	wire, err := proto.Marshal(file)
	if err != nil {
		panic(err)
	}
	var out bytes.Buffer
	writer := gzip.NewWriter(&out)
	if _, err := writer.Write(wire); err != nil {
		panic(err)
	}
	if err := writer.Close(); err != nil {
		panic(err)
	}
	return out.Bytes()
}()

func TestStandardGroupOneof(t *testing.T) {
	for _, name := range []string{"groupliketype", "GroupLikeType"} {
		var message groupOneofProbe
		if err := UnmarshalString(name+` { value: 9 }`, &message); err != nil {
			t.Fatal(err)
		}
		variant, ok := message.Choice.(*groupVariant)
		if !ok || variant.Group == nil || variant.Group.Value != 9 {
			t.Fatal("oneof group lost")
		}
		text, err := (&Marshaler{}).MarshalToString(&message)
		if err != nil || !strings.Contains(text, "GroupLikeType") {
			t.Fatalf("oneof output %q %v", text, err)
		}
		if err := UnmarshalString(text, new(groupOneofProbe)); err != nil {
			t.Fatal(err)
		}
	}
	if err := UnmarshalString(`GroupLikeType {} groupliketype {}`, new(groupOneofProbe)); err == nil {
		t.Fatal("oneof alias overwrote selection")
	}
}
