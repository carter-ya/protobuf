package generator

import d "github.com/gogo/protobuf/protoc-gen-gogo/descriptor"

func fileNeedsEditionExtensionPolicy(file *d.FileDescriptorProto) bool {
	var visit func([]*d.DescriptorProto) bool
	visit = func(messages []*d.DescriptorProto) bool {
		for _, message := range messages {
			if message.HasExtension() || len(message.Extension) > 0 || visit(message.NestedType) {
				return true
			}
		}
		return false
	}
	return len(file.Extension) > 0 || visit(file.MessageType)
}
