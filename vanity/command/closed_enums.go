package command

import (
	"github.com/gogo/protobuf/gogoproto"
	d "github.com/gogo/protobuf/protoc-gen-gogo/descriptor"
	p "github.com/gogo/protobuf/protoc-gen-gogo/plugin"
	"github.com/gogo/protobuf/vanity"
	"strings"
)

// Apply before convenience backends install legacy defaults. Explicit source
// options remain authoritative and incompatible representations are diagnosed.
func prepareClosedEnumPolicy(req *p.CodeGeneratorRequest) {
	enabled := false
	for _, part := range strings.Split(req.GetParameter(), ",") {
		kv := strings.SplitN(part, "=", 2)
		if len(kv) == 2 && kv[0] == "enforce_closed_enums" {
			enabled = kv[1] == "true"
		}
	}
	if !enabled {
		return
	}
	files := vanity.FilterFiles(req.ProtoFile, func(file *d.FileDescriptorProto) bool {
		return vanity.NotGoogleProtobufDescriptorProto(file) && file.GetSyntax() != "editions"
	})
	for _, file := range files {
		vanity.SetBoolFileOption(gogoproto.E_GoprotoUnrecognizedAll, true)(file)
	}
	vanity.ForEachFieldInFiles(files, func(field *d.FieldDescriptorProto) {
		if field.GetType() == d.FieldDescriptorProto_TYPE_ENUM && !field.IsRepeated() && !field.IsRealOneof() {
			vanity.SetBoolFieldOption(gogoproto.E_Nullable, true)(field)
		}
	})
}
