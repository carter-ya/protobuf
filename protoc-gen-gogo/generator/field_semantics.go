package generator

import (
	"github.com/gogo/protobuf/gogoproto"
	d "github.com/gogo/protobuf/protoc-gen-gogo/descriptor"
	"strconv"
	"strings"
)

// IsEditions distinguishes the private resolved generator view from legacy
// proto2/proto3 behavior. Plugins must not infer presence from syntax alone.
func (f *FileDescriptor) IsEditions() bool { return f.edition != nil }

// FieldHasPresence reports the schema's resolved presence contract. Legacy
// custom Go representations retain their separate nullable decisions.
func (g *Generator) FieldHasPresence(file *FileDescriptor, field *d.FieldDescriptorProto) bool {
	if field.IsRepeated() {
		return false
	}
	if file.edition != nil {
		return file.edition.fields[field].HasPresence()
	}
	return field.IsMessage() || field.IsRealOneof() || field.IsRequired() || field.GetProto3Optional() || !gogoproto.IsProto3(file.FileDescriptorProto)
}

// FieldProto3Value preserves the caller's historical decision for legacy files,
// and selects value/zero omission independently of UTF-8 in Editions.
func (g *Generator) FieldProto3Value(file *FileDescriptor, field *d.FieldDescriptorProto, legacy bool) bool {
	if file.edition != nil {
		return !g.FieldHasPresence(file, field)
	}
	return legacy && !field.GetProto3Optional()
}
func (g *Generator) FieldRequiresUTF8(file *FileDescriptor, field *d.FieldDescriptorProto) bool {
	if field.GetType() != d.FieldDescriptorProto_TYPE_STRING {
		return false
	}
	if file.edition != nil {
		return file.edition.fields[field].VerifyUTF8()
	}
	return gogoproto.IsProto3(file.FileDescriptorProto)
}
func (g *Generator) FieldIsDelimited(file *FileDescriptor, field *d.FieldDescriptorProto) bool {
	if file.edition != nil {
		return field.GetType() == d.FieldDescriptorProto_TYPE_GROUP
	}
	return g.IsGroup(field)
}

// FieldClosedEnumValues uses the enum definition's scope, including imports.
func (g *Generator) FieldClosedEnumValues(file *FileDescriptor, field *d.FieldDescriptorProto) []int32 {
	if file.edition == nil || field.GetType() != d.FieldDescriptorProto_TYPE_ENUM {
		return nil
	}
	encoded := file.edition.enums[field.GetTypeName()].closed
	if encoded == "" {
		return nil
	}
	var values []int32
	for _, part := range strings.Split(encoded, ":") {
		n, _ := strconv.ParseInt(part, 10, 32)
		values = append(values, int32(n))
	}
	return values
}
