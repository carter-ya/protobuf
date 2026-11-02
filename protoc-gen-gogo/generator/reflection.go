package generator

import (
	"fmt"
	"strings"

	"github.com/gogo/protobuf/gogoproto"
	"github.com/gogo/protobuf/proto"
	d "github.com/gogo/protobuf/protoc-gen-gogo/descriptor"
	p "github.com/gogo/protobuf/protoc-gen-gogo/plugin"
)

func apiFeature(features *d.FeatureSet, inherited int32) (int32, error) {
	if features == nil || !proto.HasExtension(features, goFeatureExtension) {
		return inherited, nil
	}
	f, err := readGoFeatures(features)
	if err != nil {
		return 0, err
	}
	if f.ApiLevel != nil && *f.ApiLevel != 0 {
		return int32(*f.ApiLevel), nil
	}
	return inherited, nil
}
func reflectionParameters(parameter string) (map[string]string, error) {
	params := make(map[string]string)
	for _, part := range strings.Split(parameter, ",") {
		kv := strings.SplitN(part, "=", 2)
		if len(kv) == 2 {
			params[kv[0]] = kv[1]
		}
	}
	if v, ok := params["proto_reflect"]; ok && v != "true" && v != "false" {
		return nil, fmt.Errorf("proto_reflect must be true or false")
	}
	for key, v := range params {
		if key != "default_api_level" && !strings.HasPrefix(key, "apilevelM") {
			continue
		}
		if key == "apilevelM" {
			return nil, fmt.Errorf("apilevelM must name a proto file")
		}
		if v != "API_OPEN" && v != "API_HYBRID" && v != "API_OPAQUE" {
			return nil, fmt.Errorf("%s supports API_OPEN, API_HYBRID or API_OPAQUE", key)
		}
	}
	return params, nil
}

// PrepareReflectionRequest runs before convenience backends install legacy
// defaults. Only selected messages receive native-storage defaults; explicit
// schema settings remain authoritative and receive precise diagnostics later.
func PrepareReflectionRequest(req *p.CodeGeneratorRequest) error {
	params, err := reflectionParameters(req.GetParameter())
	if err != nil {
		return err
	}
	cli := params["proto_reflect"] == "true" || (params["default_api_level"] == "API_HYBRID" || params["default_api_level"] == "API_OPAQUE")
	targets := make(map[string]bool)
	for _, n := range req.FileToGenerate {
		targets[n] = true
	}
	for _, file := range req.ProtoFile {
		if !targets[file.GetName()] {
			continue
		}
		fileDefault := defaultAPI(params, file.GetName())
		fileAPI, err := apiFeature(file.GetOptions().GetFeatures(), 0)
		if err != nil {
			return err
		}
		var visit func([]*d.DescriptorProto, int32) error
		visit = func(messages []*d.DescriptorProto, inherited int32) error {
			for _, m := range messages {
				api, err := apiFeature(m.GetOptions().GetFeatures(), inherited)
				if err != nil {
					return err
				}
				hybrid := api == 2 || api == 3 || (api == 0 && (fileDefault == 2 || fileDefault == 3))
				selected := params["proto_reflect"] == "true" || hybrid
				cli = cli || selected
				if selected && !m.GetOptions().GetMapEntry() {
					if m.Options == nil {
						m.Options = &d.MessageOptions{}
					}
					if !proto.HasExtension(m.Options, gogoproto.E_GoprotoUnrecognized) && !proto.HasExtension(file.Options, gogoproto.E_GoprotoUnrecognizedAll) {
						proto.SetExtension(m.Options, gogoproto.E_GoprotoUnrecognized, proto.Bool(true))
					}
					for _, f := range m.Field {
						if file.GetSyntax() == "editions" {
							continue
						}
						if f.Options == nil {
							f.Options = &d.FieldOptions{}
						}
						if (f.IsMessage() || f.GetType() == d.FieldDescriptorProto_TYPE_GROUP || f.GetProto3Optional() || (!gogoproto.IsProto3(file) && !f.IsRepeated() && !f.IsRealOneof() && !f.IsBytes())) && !proto.HasExtension(f.Options, gogoproto.E_Nullable) {
							proto.SetExtension(f.Options, gogoproto.E_Nullable, proto.Bool(true))
						}
					}
				}
				if err := visit(m.NestedType, api); err != nil {
					return err
				}
			}
			return nil
		}
		if err := visit(file.MessageType, fileAPI); err != nil {
			return err
		}
	}
	if cli {
		for _, key := range []string{"enforce_closed_enums", "merge_required_fields"} {
			if value, ok := params[key]; ok && value == "false" {
				return fmt.Errorf("%s=false conflicts with native reflection", key)
			}
			if params[key] != "true" {
				req.Parameter = proto.String(req.GetParameter() + "," + key + "=true")
			}
		}
	}
	return nil
}
func defaultAPI(params map[string]string, filename string) int32 {
	value := params["default_api_level"]
	if selected, ok := params["apilevelM"+filename]; ok {
		value = selected
	}
	switch value {
	case "API_HYBRID":
		return 2
	case "API_OPAQUE":
		return 3
	}
	return 1
}
func (g *Generator) apiLevel(message *Descriptor) int32 {
	level := int32(0)
	if message.File().edition != nil {
		level = message.File().edition.apiLevels[message.DescriptorProto]
	}
	if level == 0 {
		level = defaultAPI(g.Param, message.File().GetName())
	}
	if g.OpaqueAlternate && level == 2 {
		return 3
	}
	return level
}
func (g *Generator) Hybrid(message *Descriptor) bool { return g.apiLevel(message) == 2 }
func (g *Generator) Opaque(message *Descriptor) bool { return g.apiLevel(message) == 3 }
func (g *Generator) ReflectMessage(message *Descriptor) bool {
	return !message.GetOptions().GetMapEntry() && (g.Param["proto_reflect"] == "true" || g.Hybrid(message) || g.Opaque(message))
}
func (g *Generator) validateReflection(file *FileDescriptor) {
	for _, m := range file.Messages() {
		if !g.ReflectMessage(m) {
			continue
		}
		fail := func(reason string) { g.Fail("native reflection: ", file.GetName(), ":", m.GetName(), ":", reason) }
		if !gogoproto.ImportsGoGoProto(file.FileDescriptorProto) {
			fail("requires the gogo runtime")
		}
		if !gogoproto.HasTypeDecl(file.FileDescriptorProto, m.DescriptorProto) {
			fail("requires generated native type declarations")
		}
		if !gogoproto.HasUnrecognized(file.FileDescriptorProto, m.DescriptorProto) {
			fail("requires unknown-field storage")
		}
		if m.GetOptions().GetMessageSetWireFormat() {
			fail("MessageSet is unsupported")
		}
		if len(m.ExtensionRange) > 0 && !gogoproto.HasExtensionsMap(file.FileDescriptorProto, m.DescriptorProto) {
			fail("requires map-backed extensions")
		}
		if (g.Hybrid(m) || g.Opaque(m)) && !gogoproto.HasGoGetters(file.FileDescriptorProto, m.DescriptorProto) {
			if g.Opaque(m) {
				fail("OPAQUE requires getters")
			} else {
				fail("HYBRID requires getters")
			}
		}
		for _, f := range m.Field {
			if gogoproto.IsCustomType(f) || gogoproto.IsStdType(f) || gogoproto.IsEmbed(f) {
				fail("custom/std/embedded mapping is unsupported: " + f.GetName())
			}
			if f.GetOptions().GetWeak() {
				fail("weak fields are unsupported")
			}
			if (f.IsMessage() || f.GetType() == d.FieldDescriptorProto_TYPE_GROUP || (!f.IsRepeated() && !f.IsRealOneof() && !f.IsBytes() && g.FieldHasPresence(file, f))) && !gogoproto.IsNullable(f) {
				fail("requires independent native storage: " + f.GetName())
			}
		}
	}
}
