package generator

import (
	"github.com/gogo/protobuf/gogoproto"
	"github.com/gogo/protobuf/proto"
	d "github.com/gogo/protobuf/protoc-gen-gogo/descriptor"
	p "github.com/gogo/protobuf/protoc-gen-gogo/plugin"
	"testing"
)

func TestResolvedPluginContract(t *testing.T) {
	optional := d.FieldDescriptorProto_LABEL_OPTIONAL.Enum()
	file := &d.FileDescriptorProto{Name: proto.String("features.proto"), Syntax: proto.String("editions"), Edition: d.Edition_EDITION_2023.Enum(), MessageType: []*d.DescriptorProto{{Name: proto.String("Sample"), Field: []*d.FieldDescriptorProto{
		{Name: proto.String("explicit"), Number: proto.Int32(1), Label: optional, Type: d.FieldDescriptorProto_TYPE_STRING.Enum()},
		{Name: proto.String("implicit"), Number: proto.Int32(2), Label: optional, Type: d.FieldDescriptorProto_TYPE_STRING.Enum(), Options: &d.FieldOptions{Features: &d.FeatureSet{FieldPresence: d.FeatureSet_IMPLICIT.Enum(), Utf8Validation: d.FeatureSet_NONE.Enum()}}},
	}}}}
	g := New()
	g.Request = &p.CodeGeneratorRequest{FileToGenerate: []string{"features.proto"}, ProtoFile: []*d.FileDescriptorProto{file}}
	g.WrapTypes()
	resolved := g.allFiles[0]
	first, second := resolved.MessageType[0].Field[0], resolved.MessageType[0].Field[1]
	if !resolved.IsEditions() || !g.FieldHasPresence(resolved, first) || g.FieldHasPresence(resolved, second) || !g.FieldRequiresUTF8(resolved, first) || g.FieldRequiresUTF8(resolved, second) {
		t.Fatal("independent presence and UTF-8 features lost")
	}
	if g.FieldProto3Value(resolved, first, true) || !g.FieldProto3Value(resolved, second, false) {
		t.Fatal("value representation ignored resolved presence")
	}
}
func TestEditionOptionAllowlist(t *testing.T) {
	for _, tc := range []struct {
		extension *proto.ExtensionDesc
		value     interface{}
		allowed   bool
	}{
		{gogoproto.E_Customname, proto.String("Value"), true},
		{gogoproto.E_GostringAll, proto.Bool(true), true},
		{gogoproto.E_GoprotoUnrecognizedAll, proto.Bool(true), true},
		{gogoproto.E_GoprotoUnrecognizedAll, proto.Bool(false), false},
		{gogoproto.E_MarshalerAll, proto.Bool(true), false},
		{gogoproto.E_EnumvalueCustomname, proto.String("Name"), true},
	} {
		var message proto.Message
		switch tc.extension.ExtendedType.(type) {
		case *d.FileOptions:
			message = new(d.FileOptions)
		case *d.FieldOptions:
			message = new(d.FieldOptions)
		case *d.EnumValueOptions:
			message = new(d.EnumValueOptions)
		}
		if err := proto.SetExtension(message, tc.extension, tc.value); err != nil {
			t.Fatal(err)
		}
		err := validateEditionOption(message, tc.extension)
		if (err == nil) != tc.allowed {
			t.Fatalf("%d: %v", tc.extension.Field, err)
		}
	}
}
