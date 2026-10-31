package generator

import (
	"github.com/gogo/protobuf/gogoproto"
	"github.com/gogo/protobuf/protoc-gen-gogo/descriptor"
)

// MergeOneofMessage is an explicit generation policy. Historical proto2/proto3
// bindings keep replacement behavior unless regenerated with this parameter.
func (g *Generator) MergeOneofMessage(field *descriptor.FieldDescriptorProto) bool {
	return g.Param["merge_oneof_messages"] == "true" && field.IsRealOneof() && field.IsMessage()
}

func (g *Generator) validateOneofMessageMerging(file *FileDescriptor) bool {
	required := false
	var visit func([]*descriptor.DescriptorProto)
	visit = func(messages []*descriptor.DescriptorProto) {
		for _, message := range messages {
			for _, field := range message.Field {
				if !g.MergeOneofMessage(field) {
					continue
				}
				required = true
				if !gogoproto.ImportsGoGoProto(file.FileDescriptorProto) {
					g.Fail("merge_oneof_messages requires the gogo runtime: ", file.GetName(), ":", message.GetName(), ".", field.GetName())
				}
				if gogoproto.IsCustomType(field) || gogoproto.IsCastType(field) || gogoproto.IsStdType(field) {
					g.Fail("merge_oneof_messages does not support custom, cast or standard message mappings: ", file.GetName(), ":", message.GetName(), ".", field.GetName())
				}
			}
			visit(message.NestedType)
		}
	}
	visit(file.MessageType)
	return required
}
