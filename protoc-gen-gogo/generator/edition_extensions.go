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

// NeedsDeferredEditionExtensions reports extension-bearing ordinary dependency
// graphs. Their logical message boundary is the outer decoder call, rather than
// each occurrence of a child field. A plain extensible root can keep its fast
// field decoder because its own extension check already runs at that boundary.
func (g *Generator) NeedsDeferredEditionExtensions(message *Descriptor) bool {
	seen := make(map[*Descriptor]bool)
	var visit func(*Descriptor) bool
	visit = func(m *Descriptor) bool {
		if m.DescriptorProto.HasExtension() {
			return true
		}
		if seen[m] {
			return false
		}
		seen[m] = true
		for _, field := range m.Field {
			if field.IsMessage() || field.GetType() == d.FieldDescriptorProto_TYPE_GROUP {
				object := g.ObjectNamed(field.GetTypeName())
				if imported, ok := object.(*ImportedDescriptor); ok {
					object = imported.o
				}
				if visit(object.(*Descriptor)) {
					return true
				}
			}
		}
		return false
	}
	for _, field := range message.Field {
		if field.IsMessage() || field.GetType() == d.FieldDescriptorProto_TYPE_GROUP {
			object := g.ObjectNamed(field.GetTypeName())
			if imported, ok := object.(*ImportedDescriptor); ok {
				object = imported.o
			}
			if visit(object.(*Descriptor)) {
				return true
			}
		}
	}
	return false
}

func (g *Generator) fileNeedsDeferredEditionExtensions(file *FileDescriptor) bool {
	for _, message := range file.Messages() {
		if g.NeedsDeferredEditionExtensions(message) {
			return true
		}
	}
	return false
}
