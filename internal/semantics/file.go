package semantics

import (
	"fmt"
	"github.com/gogo/protobuf/internal/fieldmeta"
	d "github.com/gogo/protobuf/protoc-gen-gogo/descriptor"
)

type File struct {
	Edition  d.Edition
	Defaults fieldmeta.Features
	Fields   map[*d.FieldDescriptorProto]fieldmeta.Features
	Enums    map[*d.EnumDescriptorProto]fieldmeta.Features
}

func ResolveFile(file *d.FileDescriptorProto) (*File, error) {
	edition := file.GetEdition()
	switch file.GetSyntax() {
	case "", "proto2":
		edition = d.Edition_EDITION_PROTO2
	case "proto3":
		edition = d.Edition_EDITION_PROTO3
	case "editions":
	default:
		return nil, fmt.Errorf("%s: unknown syntax %q", file.GetName(), file.GetSyntax())
	}
	r, err := New(edition)
	if err != nil {
		return nil, fmt.Errorf("%s: %v", file.GetName(), err)
	}
	base, err := r.Resolve(r.Defaults(), file.GetOptions().GetFeatures(), d.FieldOptions_TARGET_TYPE_FILE)
	if err != nil {
		return nil, fmt.Errorf("%s: %v", file.GetName(), err)
	}
	out := &File{Edition: edition, Defaults: base, Fields: make(map[*d.FieldDescriptorProto]fieldmeta.Features), Enums: make(map[*d.EnumDescriptorProto]fieldmeta.Features)}
	resolveField := func(f *d.FieldDescriptorProto, parent fieldmeta.Features, scope string) error {
		state, err := r.Resolve(parent, f.GetOptions().GetFeatures(), d.FieldOptions_TARGET_TYPE_FIELD)
		if err != nil {
			return fmt.Errorf("%s: %s.%s: %v", file.GetName(), scope, f.GetName(), err)
		}
		out.Fields[f] = ForField(state, f)
		return nil
	}
	resolveEnums := func(enums []*d.EnumDescriptorProto, parent fieldmeta.Features, scope string) error {
		for _, e := range enums {
			state, err := r.Resolve(parent, e.GetOptions().GetFeatures(), d.FieldOptions_TARGET_TYPE_ENUM)
			if err != nil {
				return fmt.Errorf("%s: %s.%s: %v", file.GetName(), scope, e.GetName(), err)
			}
			out.Enums[e] = state
		}
		return nil
	}
	var visit func([]*d.DescriptorProto, fieldmeta.Features, string) error
	visit = func(messages []*d.DescriptorProto, parent fieldmeta.Features, scope string) error {
		for _, m := range messages {
			name := scope + "." + m.GetName()
			state, err := r.Resolve(parent, m.GetOptions().GetFeatures(), d.FieldOptions_TARGET_TYPE_MESSAGE)
			if err != nil {
				return fmt.Errorf("%s: %s: %v", file.GetName(), name, err)
			}
			oneofs := make([]fieldmeta.Features, len(m.OneofDecl))
			for i, o := range m.OneofDecl {
				oneofs[i], err = r.Resolve(state, o.GetOptions().GetFeatures(), d.FieldOptions_TARGET_TYPE_ONEOF)
				if err != nil {
					return fmt.Errorf("%s: %s.%s: %v", file.GetName(), name, o.GetName(), err)
				}
			}
			for _, f := range m.Field {
				inherited := state
				if f.OneofIndex != nil {
					if *f.OneofIndex < 0 || int(*f.OneofIndex) >= len(oneofs) {
						return fmt.Errorf("%s: %s.%s: invalid oneof index", file.GetName(), name, f.GetName())
					}
					inherited = oneofs[*f.OneofIndex]
				}
				if err := resolveField(f, inherited, name); err != nil {
					return err
				}
			}
			// Extensions inherit their declaration scope, not the extendee.
			for _, f := range m.Extension {
				if err := resolveField(f, state, name); err != nil {
					return err
				}
			}
			if err := resolveEnums(m.EnumType, state, name); err != nil {
				return err
			}
			if err := visit(m.NestedType, state, name); err != nil {
				return err
			}
		}
		return nil
	}
	if err := resolveEnums(file.EnumType, base, file.GetPackage()); err != nil {
		return nil, err
	}
	for _, f := range file.Extension {
		if err := resolveField(f, base, file.GetPackage()); err != nil {
			return nil, err
		}
	}
	if err := visit(file.MessageType, base, file.GetPackage()); err != nil {
		return nil, err
	}
	return out, nil
}
