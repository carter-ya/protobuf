package generator

import (
	"github.com/gogo/protobuf/gogoproto"
	d "github.com/gogo/protobuf/protoc-gen-gogo/descriptor"
)

func (g *Generator) requiredMergeEnabled(file *FileDescriptor) bool {
	return file.IsEditions() || g.Param["merge_required_fields"] == "true"
}

func (g *Generator) requiredDependency(field *d.FieldDescriptorProto) *Descriptor {
	object := g.ObjectNamed(field.GetTypeName())
	if imported, ok := object.(*ImportedDescriptor); ok {
		object = imported.o
	}
	return object.(*Descriptor)
}

func (g *Generator) hasRequiredGraph(message *Descriptor) bool {
	seen := make(map[*Descriptor]bool)
	var visit func(*Descriptor) bool
	visit = func(m *Descriptor) bool {
		if seen[m] {
			return false
		}
		seen[m] = true
		for _, field := range m.Field {
			if field.IsRequired() {
				return true
			}
			if (field.IsMessage() || field.GetType() == d.FieldDescriptorProto_TYPE_GROUP) && visit(g.requiredDependency(field)) {
				return true
			}
		}
		return false
	}
	return visit(message)
}

// MergeRequiredFields selects final logical-message initialization checks. The
// metadata is explicit for legacy files and automatic for Editions.
func (g *Generator) MergeRequiredFields(message *Descriptor) bool {
	return g.requiredMergeEnabled(message.File()) && g.hasRequiredGraph(message)
}

func (g *Generator) validateRequiredMerging(file *FileDescriptor) bool {
	if !g.requiredMergeEnabled(file) {
		return false
	}
	needed := false
	roots := append([]*Descriptor(nil), file.Messages()...)
	if file.IsEditions() {
		var extensions func([]*d.FieldDescriptorProto)
		extensions = func(fields []*d.FieldDescriptorProto) {
			for _, field := range fields {
				if field.IsMessage() || field.GetType() == d.FieldDescriptorProto_TYPE_GROUP {
					roots = append(roots, g.requiredDependency(field))
				}
			}
		}
		extensions(file.Extension)
		for _, message := range file.Messages() {
			extensions(message.Extension)
		}
	}
	for _, message := range roots {
		if !g.hasRequiredGraph(message) {
			continue
		}
		needed = true
		if !gogoproto.ImportsGoGoProto(file.FileDescriptorProto) {
			g.Fail("merge_required_fields requires the gogo runtime: ", file.GetName(), ":", message.GetName())
		}
		seen := make(map[*Descriptor]bool)
		var visit func(*Descriptor)
		visit = func(m *Descriptor) {
			if seen[m] {
				return
			}
			seen[m] = true
			for _, field := range m.Field {
				if field.IsRequired() && (!field.IsBytes() || gogoproto.IsCustomType(field)) && !gogoproto.IsNullable(field) {
					g.Fail("merge_required_fields requires independent required presence: ", m.File().GetName(), ":", m.GetName(), ".", field.GetName())
				}
				if field.IsMessage() || field.GetType() == d.FieldDescriptorProto_TYPE_GROUP {
					child := g.requiredDependency(field)
					if g.hasRequiredGraph(child) && !gogoproto.IsNullable(field) && !field.IsRepeated() && !m.GetOptions().GetMapEntry() {
						g.Fail("merge_required_fields requires independent message presence: ", m.File().GetName(), ":", m.GetName(), ".", field.GetName())
					}
					if g.hasRequiredGraph(child) && (gogoproto.IsCustomType(field) || gogoproto.IsCastType(field) || gogoproto.IsCastValue(field) || gogoproto.IsStdType(field)) {
						g.Fail("merge_required_fields cannot inspect mapped required message fields: ", m.File().GetName(), ":", m.GetName(), ".", field.GetName())
					}
					visit(child)
				}
			}
		}
		visit(message)
	}
	return needed
}
