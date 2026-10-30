package generator

import d "github.com/gogo/protobuf/protoc-gen-gogo/descriptor"

func fileNeedsWideRequired(file *d.FileDescriptorProto) bool {
	var visit func([]*d.DescriptorProto) bool
	visit = func(messages []*d.DescriptorProto) bool {
		for _, m := range messages {
			count := 0
			for _, field := range m.Field {
				if field.IsRequired() {
					count++
				}
			}
			if count > 64 || visit(m.NestedType) {
				return true
			}
		}
		return false
	}
	return visit(file.MessageType)
}
