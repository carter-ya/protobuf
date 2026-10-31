package generator

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/gogo/protobuf/gogoproto"
	"github.com/gogo/protobuf/internal/fieldmeta"
	"github.com/gogo/protobuf/internal/semantics"
	"github.com/gogo/protobuf/proto"
	d "github.com/gogo/protobuf/protoc-gen-gogo/descriptor"
)

type editionEnum struct {
	closed string
	first  string
}
type editionMetadata struct {
	fields       map[*d.FieldDescriptorProto]fieldmeta.Features
	enums        map[string]editionEnum
	legacyJSON   map[*d.EnumDescriptorProto]bool
	enumPrefixes map[*d.EnumValueDescriptorProto]int32
}

// Collect enums from their definition scopes, including imported proto2 enums.
// The referencing field's enum_type must never override the defining enum.
func editionEnums(files []*d.FileDescriptorProto) (map[string]editionEnum, error) {
	out := make(map[string]editionEnum)
	for _, file := range files {
		state, err := semantics.ResolveFile(file)
		if err != nil {
			return nil, err
		}
		var visit func([]*d.EnumDescriptorProto, []*d.DescriptorProto, string)
		visit = func(enums []*d.EnumDescriptorProto, messages []*d.DescriptorProto, scope string) {
			for _, e := range enums {
				info := editionEnum{}
				if len(e.Value) > 0 {
					info.first = e.Value[0].GetName()
				}
				if state.Enums[e].ClosedEnum() {
					values := make(map[int32]bool)
					for _, v := range e.Value {
						values[v.GetNumber()] = true
					}
					numbers := make([]int, 0, len(values))
					for n := range values {
						numbers = append(numbers, int(n))
					}
					sort.Ints(numbers)
					parts := make([]string, len(numbers))
					for i, n := range numbers {
						parts[i] = strconv.Itoa(n)
					}
					info.closed = strings.Join(parts, ":")
				}
				out[scope+"."+e.GetName()] = info
			}
			for _, m := range messages {
				visit(m.EnumType, m.NestedType, scope+"."+m.GetName())
			}
		}
		scope := ""
		if file.GetPackage() != "" {
			scope = "." + file.GetPackage()
		}
		visit(file.EnumType, file.MessageType, scope)
	}
	return out, nil
}

func prepareEdition(raw *d.FileDescriptorProto, enums map[string]editionEnum) (*d.FileDescriptorProto, *editionMetadata, error) {
	if raw.GetEdition() != d.Edition_EDITION_2023 && raw.GetEdition() != d.Edition_EDITION_2024 && raw.GetEdition() != d.Edition_EDITION_2026 {
		return nil, nil, fmt.Errorf("%s: unsupported edition %d", raw.GetName(), raw.GetEdition())
	}
	if !gogoproto.ImportsGoGoProto(raw) {
		return nil, nil, fmt.Errorf("%s: Editions requires the gogo runtime", raw.GetName())
	}
	// Work on a private descriptor graph; emitted reflection data uses raw.
	file := proto.Clone(raw).(*d.FileDescriptorProto)
	state, err := semantics.ResolveFile(file)
	if err != nil {
		return nil, nil, err
	}
	checkOptions := func(options proto.Message, features *d.FeatureSet, scope string) error {
		if options != nil && !reflectNilMessage(options) {
			extensions, _ := proto.ExtensionDescs(options)
			for _, e := range extensions {
				if err := validateEditionOption(options, e); err != nil {
					return fmt.Errorf("%s: %s: %v", file.GetName(), scope, err)
				}
			}
		}
		if features != nil {
			extensions, _ := proto.ExtensionDescs(features)
			for _, e := range extensions {
				if e.Field == 1002 {
					if err := validateGoFeatures(features); err != nil {
						return fmt.Errorf("%s: %s: %v", file.GetName(), scope, err)
					}
				}
			}
			if len(features.XXX_unrecognized) > 0 {
				return fmt.Errorf("%s: %s: unknown core features", file.GetName(), scope)
			}
		}
		return nil
	}
	if err := checkOptions(file.Options, file.GetOptions().GetFeatures(), file.GetName()); err != nil {
		return nil, nil, err
	}
	out := &editionMetadata{fields: state.Fields, enums: enums, legacyJSON: make(map[*d.EnumDescriptorProto]bool), enumPrefixes: make(map[*d.EnumValueDescriptorProto]int32)}
	fileLegacyJSON, err := enumLegacyJSON(file.GetOptions().GetFeatures(), false)
	if err != nil {
		return nil, nil, err
	}
	fileEnumPrefix, err := enumPrefixFeature(file.GetOptions().GetFeatures(), 1)
	if err != nil {
		return nil, nil, err
	}
	prepareExtensions := func(fields []*d.FieldDescriptorProto, scope string) error {
		for _, f := range fields {
			if err := checkOptions(f.Options, f.GetOptions().GetFeatures(), scope+"."+f.GetName()); err != nil {
				return err
			}
			features := state.Fields[f]
			if features.Required() {
				return fmt.Errorf("%s: %s: required extensions are invalid", file.GetName(), f.GetName())
			}
			if f.Options == nil {
				f.Options = &d.FieldOptions{}
			}
			if f.IsRepeated() && IsScalar(f) {
				f.Options.Packed = proto.Bool(features.Packed())
			}
			if f.IsMessage() && features.Delimited() {
				f.Type = d.FieldDescriptorProto_TYPE_GROUP.Enum()
			}
			if f.GetType() == d.FieldDescriptorProto_TYPE_ENUM && f.DefaultValue == nil && features.HasPresence() {
				if e, ok := enums[f.GetTypeName()]; ok && e.first != "" {
					f.DefaultValue = proto.String(e.first)
				}
			}
		}
		return nil
	}
	if err := prepareExtensions(file.Extension, file.GetPackage()); err != nil {
		return nil, nil, err
	}
	checkEnum := func(e *d.EnumDescriptorProto, scope string) error {
		if err := checkOptions(e.Options, e.GetOptions().GetFeatures(), scope); err != nil {
			return err
		}
		value, err := enumLegacyJSON(e.GetOptions().GetFeatures(), fileLegacyJSON)
		if err != nil {
			return err
		}
		out.legacyJSON[e] = value
		enumPrefix, err := enumPrefixFeature(e.GetOptions().GetFeatures(), fileEnumPrefix)
		if err != nil {
			return err
		}
		for _, v := range e.Value {
			if err := checkOptions(v.Options, v.GetOptions().GetFeatures(), scope+"."+v.GetName()); err != nil {
				return err
			}
			mode, err := enumPrefixFeature(v.GetOptions().GetFeatures(), enumPrefix)
			if err != nil {
				return err
			}
			if mode > 1 && (gogoproto.IsEnumCustomName(e) || gogoproto.IsEnumValueCustomName(v)) {
				return fmt.Errorf("%s: %s.%s: enum prefix stripping conflicts with a custom Go enum name", file.GetName(), scope, v.GetName())
			}
			out.enumPrefixes[v] = mode
		}
		return nil
	}
	var visit func([]*d.DescriptorProto, string) error
	visit = func(messages []*d.DescriptorProto, scope string) error {
		for _, m := range messages {
			name := scope + "." + m.GetName()
			if err := checkOptions(m.Options, m.GetOptions().GetFeatures(), name); err != nil {
				return err
			}
			if err := prepareExtensions(m.Extension, name); err != nil {
				return err
			}
			for _, o := range m.OneofDecl {
				if err := checkOptions(o.Options, o.GetOptions().GetFeatures(), name+"."+o.GetName()); err != nil {
					return err
				}
			}
			for _, f := range m.Field {
				if err := checkOptions(f.Options, f.GetOptions().GetFeatures(), name+"."+f.GetName()); err != nil {
					return err
				}
				features := state.Fields[f]
				if m.GetOptions().GetMapEntry() {
					features.FieldPresence = 0
					features.MessageEncoding = 1
					state.Fields[f] = features
				}
				if f.Options == nil {
					f.Options = &d.FieldOptions{}
				}
				if features.Required() {
					f.Label = d.FieldDescriptorProto_LABEL_REQUIRED.Enum()
				}
				if !features.HasPresence() && !f.IsRepeated() && !f.IsMessage() {
					proto.SetExtension(f.Options, gogoproto.E_Nullable, proto.Bool(false))
				}
				if f.IsRepeated() && IsScalar(f) {
					f.Options.Packed = proto.Bool(features.Packed())
				}
				if f.IsMessage() && features.Delimited() && !f.IsRepeated() {
					f.Type = d.FieldDescriptorProto_TYPE_GROUP.Enum()
				}
				// Repeated messages also use DELIMITED, except maps (identified below).
				if f.IsMessage() && features.Delimited() && f.IsRepeated() {
					isMap := false
					for _, nested := range m.NestedType {
						if nested.GetOptions().GetMapEntry() && strings.HasSuffix(f.GetTypeName(), "."+nested.GetName()) {
							isMap = true
						}
					}
					if !isMap {
						f.Type = d.FieldDescriptorProto_TYPE_GROUP.Enum()
					}
				}
				if f.GetType() == d.FieldDescriptorProto_TYPE_ENUM && f.DefaultValue == nil && features.HasPresence() {
					if e, ok := enums[f.GetTypeName()]; ok && e.first != "" {
						f.DefaultValue = proto.String(e.first)
					}
				}
			}
			for _, e := range m.EnumType {
				if err := checkEnum(e, name+"."+e.GetName()); err != nil {
					return err
				}
			}
			if err := visit(m.NestedType, name); err != nil {
				return err
			}
		}
		return nil
	}
	for _, e := range file.EnumType {
		if err := checkEnum(e, e.GetName()); err != nil {
			return nil, nil, err
		}
	}
	if err := visit(file.MessageType, file.GetPackage()); err != nil {
		return nil, nil, err
	}
	return file, out, nil
}

// An interface containing a typed nil descriptor option is not a nil interface.
func reflectNilMessage(m proto.Message) bool {
	switch v := m.(type) {
	case *d.FileOptions:
		return v == nil
	case *d.MessageOptions:
		return v == nil
	case *d.FieldOptions:
		return v == nil
	case *d.EnumOptions:
		return v == nil
	case *d.OneofOptions:
		return v == nil
	case *d.EnumValueOptions:
		return v == nil
	}
	return false
}
