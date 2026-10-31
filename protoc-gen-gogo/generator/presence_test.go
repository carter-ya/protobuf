package generator

import (
	"strings"
	"testing"

	"github.com/gogo/protobuf/gogoproto"
	"github.com/gogo/protobuf/proto"
	d "github.com/gogo/protobuf/protoc-gen-gogo/descriptor"
)

func optionalFile() *d.FileDescriptorProto {
	return &d.FileDescriptorProto{
		Name: proto.String("presence.proto"), Syntax: proto.String("proto3"),
		MessageType: []*d.DescriptorProto{{Name: proto.String("Outer"), NestedType: []*d.DescriptorProto{{
			Name: proto.String("Sample"), OneofDecl: []*d.OneofDescriptorProto{{Name: proto.String("_value")}},
			Field: []*d.FieldDescriptorProto{{Name: proto.String("value"), Number: proto.Int32(1), Label: d.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: d.FieldDescriptorProto_TYPE_INT32.Enum(), Proto3Optional: proto.Bool(true), OneofIndex: proto.Int32(0)}},
		}}}},
	}
}

// This is an eligibility contract, not a promise that every user-provided Go
// type implements the codec interfaces required by its mapping.
func TestOptionalRepresentationContract(t *testing.T) {
	for _, tc := range []struct {
		name      string
		extension *proto.ExtensionDesc
		value     interface{}
		scope     string
		reason    string
	}{
		{"default", nil, nil, "", ""},
		{"nullable", gogoproto.E_Nullable, proto.Bool(true), "field", ""},
		{"value", gogoproto.E_Nullable, proto.Bool(false), "field", "nullable=false"},
		{"cast", gogoproto.E_Casttype, proto.String("Value"), "field", ""},
		{"cast-message", gogoproto.E_Casttype, proto.String("Message"), "field", "extension combination"},
		{"custom", gogoproto.E_Customtype, proto.String("Value"), "field", ""},
		{"custom-message", gogoproto.E_Customtype, proto.String("Value"), "field", "customtype requires a bytes field"},
		{"embed", gogoproto.E_Embed, proto.Bool(true), "field", "extension combination"},
		{"stdtime", gogoproto.E_Stdtime, proto.Bool(true), "field", ""},
		{"stdduration", gogoproto.E_Stdduration, proto.Bool(true), "field", ""},
		{"wktpointer", gogoproto.E_Wktpointer, proto.Bool(true), "field", ""},
		{"face", gogoproto.E_Face, proto.Bool(true), "message", "extension combination"},
		{"face-all", gogoproto.E_FaceAll, proto.Bool(true), "file", "extension combination"},
		{"union", gogoproto.E_Onlyone, proto.Bool(true), "message", "extension combination"},
		{"union-all", gogoproto.E_OnlyoneAll, proto.Bool(true), "file", "extension combination"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			file := optionalFile()
			message := file.MessageType[0].NestedType[0]
			field := message.Field[0]
			if tc.name == "wktpointer" || tc.name == "cast-message" {
				field.Type = d.FieldDescriptorProto_TYPE_MESSAGE.Enum()
				field.TypeName = proto.String(".google.protobuf.Int32Value")
			}
			switch tc.name {
			case "custom":
				field.Type = d.FieldDescriptorProto_TYPE_BYTES.Enum()
			case "custom-message":
				field.Type = d.FieldDescriptorProto_TYPE_MESSAGE.Enum()
				field.TypeName = proto.String(".Sample")
			case "stdtime":
				field.Type = d.FieldDescriptorProto_TYPE_MESSAGE.Enum()
				field.TypeName = proto.String(".google.protobuf.Timestamp")
			case "stdduration":
				field.Type = d.FieldDescriptorProto_TYPE_MESSAGE.Enum()
				field.TypeName = proto.String(".google.protobuf.Duration")
			}
			if tc.extension != nil {
				var options proto.Message
				switch tc.scope {
				case "field":
					field.Options = new(d.FieldOptions)
					options = field.Options
				case "message":
					message.Options = new(d.MessageOptions)
					options = message.Options
				case "file":
					file.Options = new(d.FileOptions)
					options = file.Options
				}
				if err := proto.SetExtension(options, tc.extension, tc.value); err != nil {
					t.Fatal(err)
				}
			}
			err := validateOptional(file)
			if tc.reason == "" {
				if err != nil {
					t.Fatal(err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tc.reason) || !strings.Contains(err.Error(), "presence.proto: Outer.Sample.value:") {
				t.Fatalf("expected a scoped %q diagnostic, got %v", tc.reason, err)
			}
		})
	}
}

func TestOptionalDescriptorContract(t *testing.T) {
	for _, tc := range []struct {
		name   string
		change func(*d.FileDescriptorProto, *d.DescriptorProto, *d.FieldDescriptorProto)
		reason string
	}{
		{"syntax", func(f *d.FileDescriptorProto, _ *d.DescriptorProto, _ *d.FieldDescriptorProto) {
			f.Syntax = proto.String("proto2")
		}, "requires proto3 syntax"},
		{"runtime", func(f *d.FileDescriptorProto, _ *d.DescriptorProto, _ *d.FieldDescriptorProto) {
			f.Options = new(d.FileOptions)
			proto.SetExtension(f.Options, gogoproto.E_GogoprotoImport, proto.Bool(false))
		}, "requires the gogo runtime"},
		{"repeated", func(_ *d.FileDescriptorProto, _ *d.DescriptorProto, f *d.FieldDescriptorProto) {
			f.Label = d.FieldDescriptorProto_LABEL_REPEATED.Enum()
		}, "invalid proto3 optional descriptor"},
		{"missing-oneof", func(_ *d.FileDescriptorProto, _ *d.DescriptorProto, f *d.FieldDescriptorProto) { f.OneofIndex = nil }, "invalid proto3 optional descriptor"},
		{"negative-oneof", func(_ *d.FileDescriptorProto, _ *d.DescriptorProto, f *d.FieldDescriptorProto) {
			f.OneofIndex = proto.Int32(-1)
		}, "invalid proto3 optional descriptor"},
		{"outside-oneof", func(_ *d.FileDescriptorProto, _ *d.DescriptorProto, f *d.FieldDescriptorProto) {
			f.OneofIndex = proto.Int32(1)
		}, "invalid proto3 optional descriptor"},
		{"shared-oneof", func(_ *d.FileDescriptorProto, m *d.DescriptorProto, f *d.FieldDescriptorProto) {
			sibling := proto.Clone(f).(*d.FieldDescriptorProto)
			sibling.Name = proto.String("other")
			sibling.Number = proto.Int32(2)
			sibling.Proto3Optional = nil
			m.Field = append(m.Field, sibling)
		}, "synthetic oneof must contain exactly one optional field"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			file := optionalFile()
			message := file.MessageType[0].NestedType[0]
			field := message.Field[0]
			tc.change(file, message, field)
			if err := validateOptional(file); err == nil || !strings.Contains(err.Error(), tc.reason) {
				t.Fatalf("expected %q, got %v", tc.reason, err)
			}
		})
	}
	// Validation of new optional combinations must not restrict historical files.
	file := optionalFile()
	file.MessageType[0].NestedType[0].Field[0].Proto3Optional = nil
	file.Syntax = proto.String("proto2")
	if err := validateOptional(file); err != nil {
		t.Fatal(err)
	}
}

func TestOptionalInvalidMappings(t *testing.T) {
	for _, tc := range []struct {
		name     string
		kind     d.FieldDescriptorProto_Type
		typeName string
		options  map[*proto.ExtensionDesc]interface{}
		reason   string
	}{
		{"stdtime-scalar", d.FieldDescriptorProto_TYPE_INT32, "", map[*proto.ExtensionDesc]interface{}{gogoproto.E_Stdtime: proto.Bool(true)}, "stdtime requires"},
		{"duration-wrong-message", d.FieldDescriptorProto_TYPE_MESSAGE, ".google.protobuf.Timestamp", map[*proto.ExtensionDesc]interface{}{gogoproto.E_Stdduration: proto.Bool(true)}, "stdduration requires"},
		{"wrapper-wrong-message", d.FieldDescriptorProto_TYPE_MESSAGE, ".google.protobuf.Timestamp", map[*proto.ExtensionDesc]interface{}{gogoproto.E_Wktpointer: proto.Bool(true)}, "wktpointer requires"},
		{"custom-scalar", d.FieldDescriptorProto_TYPE_INT32, "", map[*proto.ExtensionDesc]interface{}{gogoproto.E_Customtype: proto.String("Value")}, "customtype requires a bytes field"},
		{"custom-and-cast", d.FieldDescriptorProto_TYPE_BYTES, "", map[*proto.ExtensionDesc]interface{}{gogoproto.E_Customtype: proto.String("Value"), gogoproto.E_Casttype: proto.String("Value")}, "conflicting"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			file := optionalFile()
			field := file.MessageType[0].NestedType[0].Field[0]
			field.Type = tc.kind.Enum()
			field.TypeName = proto.String(tc.typeName)
			field.Options = new(d.FieldOptions)
			for extension, value := range tc.options {
				if err := proto.SetExtension(field.Options, extension, value); err != nil {
					t.Fatal(err)
				}
			}
			if err := validateOptional(file); err == nil || !strings.Contains(err.Error(), tc.reason) || !strings.Contains(err.Error(), "Outer.Sample.value:") {
				t.Fatalf("expected scoped %s, got %v", tc.reason, err)
			}
		})
	}
}

func TestOptionalUnsafeMappingRejected(t *testing.T) {
	for _, extension := range []*proto.ExtensionDesc{gogoproto.E_UnsafeMarshaler, gogoproto.E_UnsafeUnmarshaler} {
		file := optionalFile()
		message := file.MessageType[0].NestedType[0]
		field := message.Field[0]
		field.Options = new(d.FieldOptions)
		message.Options = new(d.MessageOptions)
		if err := proto.SetExtension(field.Options, gogoproto.E_Casttype, proto.String("Value")); err != nil {
			t.Fatal(err)
		}
		if err := proto.SetExtension(message.Options, extension, proto.Bool(true)); err != nil {
			t.Fatal(err)
		}
		if err := validateOptional(file); err == nil || !strings.Contains(err.Error(), "unsafe codecs") {
			t.Fatalf("unsafe mapping accepted: %v", err)
		}
	}
}
