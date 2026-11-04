package generator

import (
	"github.com/gogo/protobuf/gogoproto"
	"github.com/gogo/protobuf/proto"
	d "github.com/gogo/protobuf/protoc-gen-gogo/descriptor"
	p "github.com/gogo/protobuf/protoc-gen-gogo/plugin"
	"strings"
	"testing"
)

func TestFileAPISelectionAndScopedOverrides(t *testing.T) {
	g := New()
	g.Param = map[string]string{"default_api_level": "API_HYBRID", "apilevelMselected.proto": "API_OPAQUE"}
	m := &Descriptor{DescriptorProto: &d.DescriptorProto{Name: proto.String("Record")}, common: common{file: &FileDescriptor{FileDescriptorProto: &d.FileDescriptorProto{Name: proto.String("selected.proto")}}}}
	if !g.Opaque(m) {
		t.Fatal("file selection ignored")
	}
	m.file.Name = proto.String("other.proto")
	if !g.Hybrid(m) {
		t.Fatal("file selection leaked to other file")
	}
	g.OpaqueAlternate = true
	if !g.Opaque(m) {
		t.Fatal("HYBRID alternate remains open")
	}
	m.file.edition = &editionMetadata{apiLevels: map[*d.DescriptorProto]int32{m.DescriptorProto: 1}}
	if g.apiLevel(m) != 1 {
		t.Fatal("alternate overrode explicit OPEN")
	}
	m.file.edition.apiLevels[m.DescriptorProto] = 3
	g.OpaqueAlternate = false
	if !g.Opaque(m) {
		t.Fatal("explicit OPAQUE ignored")
	}
}

func TestFileSelectionPreparesNativeStorageAndPolicies(t *testing.T) {
	m := &d.DescriptorProto{Name: proto.String("Record"), Field: []*d.FieldDescriptorProto{{Name: proto.String("value"), Number: proto.Int32(1), Type: d.FieldDescriptorProto_TYPE_INT32.Enum(), Label: d.FieldDescriptorProto_LABEL_OPTIONAL.Enum()}}}
	req := &p.CodeGeneratorRequest{Parameter: proto.String("apilevelMselected.proto=API_OPAQUE"), FileToGenerate: []string{"selected.proto"}, ProtoFile: []*d.FileDescriptorProto{{Name: proto.String("selected.proto"), Syntax: proto.String("proto2"), MessageType: []*d.DescriptorProto{m}}}}
	if err := PrepareReflectionRequest(req); err != nil {
		t.Fatal(err)
	}
	if !gogoproto.HasUnrecognized(req.ProtoFile[0], m) || !gogoproto.IsNullable(m.Field[0]) {
		t.Fatal("private message lost native storage")
	}
	if !strings.Contains(req.GetParameter(), "enforce_closed_enums=true") || !strings.Contains(req.GetParameter(), "merge_required_fields=true") {
		t.Fatal("native policies missing")
	}
	for _, parameter := range []string{"apilevelM=API_OPAQUE", "apilevelMselected.proto=API_UNKNOWN"} {
		if _, err := reflectionParameters(parameter); err == nil {
			t.Fatal("bad file API accepted", parameter)
		}
	}
}
