package generator

import (
	"github.com/gogo/protobuf/gogoproto"
	d "github.com/gogo/protobuf/protoc-gen-gogo/descriptor"
)

func (g *Generator) EnforceClosedEnums(file *FileDescriptor) bool { return file.legacyEnums != nil }

func (g *Generator) validateClosedEnumPolicy(file *FileDescriptor) {
	if !g.EnforceClosedEnums(file) {
		return
	}
	if !gogoproto.ImportsGoGoProto(file.FileDescriptorProto) {
		g.Fail("enforce_closed_enums requires the gogo runtime: ", file.GetName())
	}
	validateField := func(scope string, field *d.FieldDescriptorProto) {
		if field.GetType() != d.FieldDescriptorProto_TYPE_ENUM || file.legacyEnums[field.GetTypeName()].closed == "" {
			return
		}
		if gogoproto.IsCustomType(field) || gogoproto.IsStdType(field) {
			g.Fail("enforce_closed_enums requires a native enum representation: ", scope, ".", field.GetName())
		}
		if !field.IsRepeated() && !field.IsRealOneof() && !gogoproto.IsNullable(field) {
			g.Fail("enforce_closed_enums requires independent enum presence: ", scope, ".", field.GetName())
		}
	}
	for _, field := range file.Extension {
		validateField(file.GetName(), field)
	}
	for _, message := range file.Messages() {
		if !message.GetOptions().GetMapEntry() && !gogoproto.HasUnrecognized(file.FileDescriptorProto, message.DescriptorProto) {
			g.Fail("enforce_closed_enums requires unknown-field storage: ", file.GetName(), ":", message.GetName())
		}
		for _, field := range message.Field {
			validateField(message.GetName(), field)
		}
		for _, field := range message.Extension {
			validateField(message.GetName(), field)
		}
	}
}
