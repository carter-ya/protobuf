package generator

import (
	"fmt"
	"github.com/gogo/protobuf/gogoproto"
	descriptor "github.com/gogo/protobuf/protoc-gen-gogo/descriptor"
)

func validateOptional(file *descriptor.FileDescriptorProto) error {
	if !file.HasProto3Optional() {
		return nil
	}
	if file.GetSyntax() != "proto3" {
		return fmt.Errorf("%s: proto3_optional requires proto3 syntax", file.GetName())
	}
	if !gogoproto.ImportsGoGoProto(file) {
		return fmt.Errorf("%s: proto3 optional currently requires the gogo runtime; use gogofast instead of gofast", file.GetName())
	}
	var visit func([]*descriptor.DescriptorProto, string) error
	visit = func(messages []*descriptor.DescriptorProto, parent string) error {
		for _, m := range messages {
			name := parent + m.GetName()
			for _, f := range m.Field {
				if !f.GetProto3Optional() {
					continue
				}
				fail := func(reason string) error {
					return fmt.Errorf("%s: %s.%s: %s", file.GetName(), name, f.GetName(), reason)
				}
				if f.IsRepeated() || f.OneofIndex == nil || *f.OneofIndex < 0 || int(*f.OneofIndex) >= len(m.OneofDecl) {
					return fail("invalid proto3 optional descriptor")
				}
				count := 0
				for _, sibling := range m.Field {
					if sibling.OneofIndex != nil && *sibling.OneofIndex == *f.OneofIndex {
						count++
					}
				}
				if count != 1 {
					return fail("synthetic oneof must contain exactly one optional field")
				}
				if !gogoproto.IsNullable(f) {
					return fail("nullable=false cannot preserve proto3 optional presence")
				}
				if gogoproto.IsCustomType(f) || gogoproto.IsCastType(f) || gogoproto.IsEmbed(f) || gogoproto.IsStdType(f) || gogoproto.IsFace(file, m) || gogoproto.IsUnion(file, m) {
					return fail("this gogo extension combination is not yet supported for proto3 optional")
				}
			}
			if err := visit(m.NestedType, name+"."); err != nil {
				return err
			}
		}
		return nil
	}
	return visit(file.MessageType, "")
}
