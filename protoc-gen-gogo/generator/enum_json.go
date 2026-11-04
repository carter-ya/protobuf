package generator

import (
	"fmt"
	"github.com/gogo/protobuf/proto"
	d "github.com/gogo/protobuf/protoc-gen-gogo/descriptor"
	"strings"
)

type enumJSONOption struct {
	CustomName       *string `protobuf:"bytes,1,opt,name=string"`
	XXX_unrecognized []byte
}

func (m *enumJSONOption) Reset()         { *m = enumJSONOption{} }
func (m *enumJSONOption) String() string { return proto.CompactTextString(m) }
func (*enumJSONOption) ProtoMessage()    {}

var enumJSONExtension = &proto.ExtensionDesc{ExtendedType: (*d.EnumValueOptions)(nil), ExtensionType: (*enumJSONOption)(nil), Field: 998, Name: "pb.enumvalue.json", Tag: "bytes,998,opt,name=json"}

func (g *Generator) registerEnumJSON(enum *EnumDescriptor, typeName string) {
	if enum.file.GetEdition() != d.Edition_EDITION_2026 {
		return
	}
	canonical := make(map[int32]bool)
	var names, values []string
	for _, value := range enum.Value {
		jsonName := value.GetName()
		if proto.HasExtension(value.Options, enumJSONExtension) {
			option, err := proto.GetExtension(value.Options, enumJSONExtension)
			if err != nil {
				g.Fail(fmt.Sprintf("%s: %s: invalid JSON enum option: %v", enum.file.GetName(), value.GetName(), err))
			}
			custom := option.(*enumJSONOption)
			if custom.CustomName != nil {
				jsonName = *custom.CustomName
			}
		}
		if !canonical[value.GetNumber()] {
			names = append(names, fmt.Sprintf("%d:%q", value.GetNumber(), jsonName))
			canonical[value.GetNumber()] = true
		}
		values = append(values, fmt.Sprintf("%q:%d", jsonName, value.GetNumber()))
		if jsonName != value.GetName() {
			values = append(values, fmt.Sprintf("%q:%d", value.GetName(), value.GetNumber()))
		}
	}
	// A JSON name can coincide with an alias spelling; protoc checks ambiguity.
	// Build initialization statements instead of a map literal to avoid duplicate
	// Go literal keys when aliases share a JSON spelling.
	g.addInitf("%s.RegisterEnumJSONNames(%q, map[int32]string{%s}, %s())", g.Pkg["proto"], typeName, strings.Join(names, ","), enumJSONValuesFunctionName(enum))
	g.P("func ", enumJSONValuesFunctionName(enum), "() map[string]int32 {")
	g.P("values := make(map[string]int32)")
	for _, entry := range values {
		parts := strings.LastIndex(entry, ":")
		g.P("values[", entry[:parts], "] = ", entry[parts+1:])
	}
	g.P("return values")
	g.P("}")
}
func enumJSONValuesFunctionName(enum *EnumDescriptor) string {
	return "enumJSONValues_" + CamelCaseSlice(enum.TypeName())
}
