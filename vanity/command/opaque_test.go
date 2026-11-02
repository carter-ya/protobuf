package command

import (
	"go/token"
	"strings"
	"testing"

	"github.com/gogo/protobuf/gogoproto"
	"github.com/gogo/protobuf/proto"
	d "github.com/gogo/protobuf/protoc-gen-gogo/descriptor"
	p "github.com/gogo/protobuf/protoc-gen-gogo/plugin"
)

func opaqueRequest(level string) *p.CodeGeneratorRequest {
	return &p.CodeGeneratorRequest{FileToGenerate: []string{"variant.proto"}, Parameter: proto.String("paths=source_relative,annotate_code=true,default_api_level=" + level), ProtoFile: []*d.FileDescriptorProto{{
		Name: proto.String("variant.proto"), Package: proto.String("variant"), Syntax: proto.String("proto3"), Options: &d.FileOptions{GoPackage: proto.String("example.com/variant")},
		MessageType: []*d.DescriptorProto{{Name: proto.String("Record"), Field: []*d.FieldDescriptorProto{{Name: proto.String("value"), Number: proto.Int32(1), Label: d.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: d.FieldDescriptorProto_TYPE_INT32.Enum()}}}},
	}}}
}

func TestHybridAlternateBuildAnnotations(t *testing.T) {
	response := Generate(opaqueRequest("API_HYBRID"))
	files := make(map[string]string)
	for _, f := range response.File {
		files[f.GetName()] = f.GetContent()
	}
	if len(files) != 4 {
		t.Fatalf("expected two Go outputs and two annotation files, got %v", files)
	}
	for _, test := range []struct{ name, expression, field string }{
		{"variant.pb.go", "!protoopaque", "Value"}, {"variant_protoopaque.pb.go", "protoopaque", "xxx_hidden_Value"},
	} {
		code := files[test.name]
		if !strings.HasPrefix(code, "//go:build "+test.expression+"\n// +build "+test.expression+"\n\n") {
			t.Fatal("missing mutually exclusive build constraints", test.name)
		}
		info := new(d.GeneratedCodeInfo)
		if err := proto.UnmarshalText(files[test.name+".meta"], info); err != nil {
			t.Fatal(err)
		}
		found := false
		for _, a := range info.Annotation {
			begin, end := int(a.GetBegin()), int(a.GetEnd())
			if begin < 0 || end <= begin || end > len(code) {
				t.Fatal("annotation outside generated code", a)
			}
			name := code[begin:end]
			if !token.IsIdentifier(name) {
				t.Fatalf("annotation moved off identifier in %s: %q", test.name, name)
			}
			if name == test.field {
				found = true
			}
		}
		if !found {
			t.Fatal("missing field annotation", test.name, test.field)
		}
	}
}

func TestNonHybridOutputsHaveNoAlternate(t *testing.T) {
	for _, level := range []string{"API_OPEN", "API_OPAQUE"} {
		r := Generate(opaqueRequest(level))
		if len(r.File) != 2 {
			t.Fatal("unexpected alternate files", level, len(r.File))
		}
		for _, f := range r.File {
			if strings.Contains(f.GetContent(), "//go:build") {
				t.Fatal("unexpected build constraint", level)
			}
		}
	}
}

func TestAlternatesRetainDeclarationOrderBeforeCodecMutation(t *testing.T) {
	req := opaqueRequest("API_HYBRID")
	f := req.ProtoFile[0]
	f.MessageType[0].Field = []*d.FieldDescriptorProto{
		{Name: proto.String("high"), Number: proto.Int32(20), Type: d.FieldDescriptorProto_TYPE_INT32.Enum(), Label: d.FieldDescriptorProto_LABEL_OPTIONAL.Enum()},
		{Name: proto.String("low"), Number: proto.Int32(1), Type: d.FieldDescriptorProto_TYPE_INT32.Enum(), Label: d.FieldDescriptorProto_LABEL_OPTIONAL.Enum()},
	}
	proto.SetExtension(f.Options, gogoproto.E_MarshalerAll, proto.Bool(true))
	proto.SetExtension(f.Options, gogoproto.E_SizerAll, proto.Bool(true))
	response := Generate(req)
	for _, file := range response.File {
		if !strings.HasSuffix(file.GetName(), ".go") {
			continue
		}
		body := strings.Split(file.GetContent(), "type Record_builder struct")[1]
		if strings.Index(body, "High") > strings.Index(body, "Low") {
			t.Fatal("codec mutation changed builder declaration order", file.GetName())
		}
	}
}
