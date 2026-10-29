package generator

import (
	"github.com/gogo/protobuf/gogoproto"
	d "github.com/gogo/protobuf/protoc-gen-gogo/descriptor"
)

func messageHasFastCodec(file *d.FileDescriptorProto, message *d.DescriptorProto) bool {
	return gogoproto.IsMarshaler(file, message) || gogoproto.IsUnmarshaler(file, message) || gogoproto.IsSizer(file, message) || gogoproto.IsProtoSizer(file, message)
}
func fileHasFastCodec(file *d.FileDescriptorProto) bool {
	var visit func([]*d.DescriptorProto) bool
	visit = func(messages []*d.DescriptorProto) bool {
		for _, message := range messages {
			if messageHasFastCodec(file, message) || visit(message.NestedType) {
				return true
			}
		}
		return false
	}
	return visit(file.MessageType)
}

// Fast methods call dependency methods directly. Diagnose partial per-message
// configurations rather than producing Go references to methods that do not exist.
func (g *Generator) validateEditionFastDependencies(file *FileDescriptor) {
	if !file.IsEditions() {
		return
	}
	for _, message := range file.Messages() {
		if message.GetOptions().GetMapEntry() {
			continue
		}
		marshaler := gogoproto.IsMarshaler(file.FileDescriptorProto, message.DescriptorProto)
		sizer := gogoproto.IsSizer(file.FileDescriptorProto, message.DescriptorProto)
		protoSizer := gogoproto.IsProtoSizer(file.FileDescriptorProto, message.DescriptorProto)
		unmarshaler := gogoproto.IsUnmarshaler(file.FileDescriptorProto, message.DescriptorProto)
		if marshaler && !sizer && !protoSizer {
			g.Fail(file.GetName(), ": ", message.GetName(), ": Editions marshaler requires sizer or protosizer")
		}
		if !marshaler && !sizer && !protoSizer && !unmarshaler {
			continue
		}
		for _, field := range message.Field {
			target := field
			if g.IsMap(field) {
				target = g.GoMapType(nil, field).ValueAliasField
			}
			if !target.IsMessage() && target.GetType() != d.FieldDescriptorProto_TYPE_GROUP {
				continue
			}
			object := g.ObjectNamed(target.GetTypeName())
			if imported, ok := object.(*ImportedDescriptor); ok {
				object = imported.o
			}
			dep := object.(*Descriptor)
			df, dm := dep.File().FileDescriptorProto, dep.DescriptorProto
			if marshaler && !gogoproto.IsMarshaler(df, dm) || sizer && !gogoproto.IsSizer(df, dm) || protoSizer && !gogoproto.IsProtoSizer(df, dm) || unmarshaler && !gogoproto.IsUnmarshaler(df, dm) {
				g.Fail(file.GetName(), ": ", message.GetName(), ".", field.GetName(), ": Editions fast dependency requires matching codec methods: ", target.GetTypeName())
			}
		}
	}
}
