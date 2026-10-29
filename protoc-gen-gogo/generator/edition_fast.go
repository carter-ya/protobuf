package generator

import (
	"fmt"
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
func validateEditionFast(file *d.FileDescriptorProto) error {
	if !fileHasFastCodec(file) {
		return nil
	}
	if len(file.Extension) > 0 {
		return fmt.Errorf("%s: fast Editions extension declarations are not yet supported", file.GetName())
	}
	var visit func([]*d.DescriptorProto) error
	visit = func(messages []*d.DescriptorProto) error {
		for _, message := range messages {
			if messageHasFastCodec(file, message) {
				if len(message.ExtensionRange) > 0 || len(message.Extension) > 0 {
					return fmt.Errorf("%s: %s: fast Editions extensions are not yet supported", file.GetName(), message.GetName())
				}
				for _, field := range message.Field {
					if field.IsRealOneof() || field.IsMessage() || field.GetType() == d.FieldDescriptorProto_TYPE_GROUP {
						return fmt.Errorf("%s: %s.%s: fast Editions complex fields are not yet supported", file.GetName(), message.GetName(), field.GetName())
					}
				}
			}
			if err := visit(message.NestedType); err != nil {
				return err
			}
		}
		return nil
	}
	return visit(file.MessageType)
}
