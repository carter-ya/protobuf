package generator

import (
	"github.com/gogo/protobuf/gogoproto"
	"github.com/gogo/protobuf/proto"
	d "github.com/gogo/protobuf/protoc-gen-gogo/descriptor"
	p "github.com/gogo/protobuf/protoc-gen-gogo/plugin"
	"testing"
)

func TestFreshGeneratorEditionEnumMap(t *testing.T) {
	options := &d.FileOptions{GoPackage: proto.String("example.com/freshmap")}
	for _, extension := range []*proto.ExtensionDesc{gogoproto.E_MarshalerAll, gogoproto.E_UnmarshalerAll, gogoproto.E_SizerAll} {
		if err := proto.SetExtension(options, extension, proto.Bool(true)); err != nil {
			t.Fatal(err)
		}
	}
	field := func(name string, tag int32, kind d.FieldDescriptorProto_Type, typeName string, label d.FieldDescriptorProto_Label) *d.FieldDescriptorProto {
		value := &d.FieldDescriptorProto{Name: proto.String(name), Number: proto.Int32(tag), Type: kind.Enum(), Label: label.Enum()}
		if typeName != "" {
			value.TypeName = proto.String(typeName)
		}
		return value
	}
	file := &d.FileDescriptorProto{Name: proto.String("fresh.proto"), Package: proto.String("fresh"), Syntax: proto.String("editions"), Edition: d.Edition_EDITION_2023.Enum(), Options: options,
		EnumType: []*d.EnumDescriptorProto{{Name: proto.String("State"), Value: []*d.EnumValueDescriptorProto{{Name: proto.String("STATE_ZERO"), Number: proto.Int32(0)}}}},
		MessageType: []*d.DescriptorProto{{Name: proto.String("Record"),
			Field: []*d.FieldDescriptorProto{field("values", 1, d.FieldDescriptorProto_TYPE_MESSAGE, ".fresh.Record.ValuesEntry", d.FieldDescriptorProto_LABEL_REPEATED)},
			NestedType: []*d.DescriptorProto{{Name: proto.String("ValuesEntry"), Options: &d.MessageOptions{MapEntry: proto.Bool(true)}, Field: []*d.FieldDescriptorProto{
				field("key", 1, d.FieldDescriptorProto_TYPE_STRING, "", d.FieldDescriptorProto_LABEL_OPTIONAL),
				field("value", 2, d.FieldDescriptorProto_TYPE_ENUM, ".fresh.State", d.FieldDescriptorProto_LABEL_OPTIONAL),
			}}},
		}},
	}
	g := New()
	bridge := &d.FileDescriptorProto{Name: proto.String("bridge.proto"), Package: proto.String("bridge"),
		Options:    &d.FileOptions{GoPackage: proto.String("example.com/freshbridge")},
		Dependency: []string{"fresh.proto"}, PublicDependency: []int32{0}}
	g.Request = &p.CodeGeneratorRequest{FileToGenerate: []string{"bridge.proto"}, ProtoFile: []*d.FileDescriptorProto{file, bridge}}
	g.CommandLineParameters("")
	g.WrapTypes()
	g.SetPackageNames()
	g.BuildTypeNameMap()
	g.GenerateAllFiles()
	if len(g.Response.File) != 1 || g.Response.File[0].GetContent() == "" {
		t.Fatal("fresh enum-map request did not generate its map")
	}
}
