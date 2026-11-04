package generator

import (
	"fmt"
	"go/token"
	"strings"

	"github.com/gogo/protobuf/gogoproto"
	"github.com/gogo/protobuf/proto"
	d "github.com/gogo/protobuf/protoc-gen-gogo/descriptor"
)

// Enum prefixes match without case or underscores. A missing prefix or an empty
// remainder keeps the original spelling, as in the official Go generator.
func trimEnumValuePrefix(value, enum string) string {
	prefix := strings.ReplaceAll(strings.ToLower(enum), "_", "")
	position := 0
	for _, letter := range prefix {
		for position < len(value) && value[position] == '_' {
			position++
		}
		if position == len(value) || strings.ToLower(value[position:position+1]) != string(letter) {
			return value
		}
		position++
	}
	trimmed := strings.TrimLeft(value[position:], "_")
	if trimmed == "" {
		return value
	}
	return trimmed
}

func enumPrefixFeature(features *d.FeatureSet, inherited int32) (int32, error) {
	if features == nil || !proto.HasExtension(features, goFeatureExtension) {
		return inherited, nil
	}
	settings, err := readGoFeatures(features)
	if err != nil {
		return 0, err
	}
	if settings.StripEnumPrefix != nil {
		return int32(*settings.StripEnumPrefix), nil
	}
	return inherited, nil
}

func (enum *EnumDescriptor) valueNames(value *d.EnumValueDescriptorProto) (name, alias string) {
	spelling := value.GetName()
	if gogoproto.IsEnumValueCustomName(value) {
		spelling = gogoproto.GetEnumValueCustomName(value)
	}
	prefix := ""
	if gogoproto.EnabledGoEnumPrefix(enum.file.FileDescriptorProto, enum.EnumDescriptorProto) {
		prefix = enum.prefix()
	}
	original := prefix + spelling
	mode := int32(1)
	if enum.file.edition != nil {
		mode = enum.file.edition.enumPrefixes[value]
	}
	if mode <= 1 {
		return original, ""
	}
	name = prefix + trimEnumValuePrefix(spelling, enum.GetName())
	if mode == 2 && name != original {
		alias = original
	}
	return name, alias
}

func (g *Generator) validateEnumNames(file *FileDescriptor) {
	var files []*FileDescriptor
	stripping := false
	for _, candidate := range g.allFiles {
		if candidate.importPath != file.importPath {
			continue
		}
		files = append(files, candidate)
		if candidate.edition != nil {
			for _, mode := range candidate.edition.enumPrefixes {
				stripping = stripping || mode > 1
			}
		}
	}
	if !stripping {
		return
	}
	seen := make(map[string]string)
	claim := func(name, scope string) {
		if !token.IsIdentifier(name) {
			g.Fail(fmt.Sprintf("%s: stripped enum Go name %s is not a valid identifier for %s", file.GetName(), name, scope))
		}
		if previous, ok := seen[name]; ok {
			g.Fail(fmt.Sprintf("%s: stripped enum Go name %s collides between %s and %s", file.GetName(), name, previous, scope))
		}
		seen[name] = scope
	}
	for _, candidate := range files {
		for _, message := range candidate.Messages() {
			if !message.GetOptions().GetMapEntry() {
				claim(CamelCaseSlice(message.TypeName()), message.GetName())
			}
		}
		for _, enum := range candidate.enum {
			claim(CamelCaseSlice(enum.alias()), enum.GetName())
			if !gogoproto.HasEnumDecl(candidate.FileDescriptorProto, enum.EnumDescriptorProto) {
				continue
			}
			for _, value := range enum.Value {
				name, alias := enum.valueNames(value)
				scope := enum.GetName() + "." + value.GetName()
				claim(name, scope)
				if alias != "" {
					claim(alias, scope)
				}
			}
		}
	}
}
