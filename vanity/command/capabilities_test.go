package command

import (
	"github.com/gogo/protobuf/gogoproto"
	"github.com/gogo/protobuf/proto"
	d "github.com/gogo/protobuf/protoc-gen-gogo/descriptor"
	"github.com/gogo/protobuf/protoc-gen-gogo/generator"
	p "github.com/gogo/protobuf/protoc-gen-gogo/plugin"
	"strings"
	"testing"
)

func request() *p.CodeGeneratorRequest {
	return &p.CodeGeneratorRequest{FileToGenerate: []string{"test.proto"}, ProtoFile: []*d.FileDescriptorProto{{Name: proto.String("test.proto"), Package: proto.String("fixture"), Options: &d.FileOptions{GoPackage: proto.String("example.com/fixture")}}}}
}
func TestProductionCapabilities(t *testing.T) {
	r := Generate(request())
	if r.GetSupportedFeatures() != 3 || r.GetMinimumEdition() != int32(d.Edition_EDITION_2023) || r.GetMaximumEdition() != int32(d.Edition_EDITION_2026) {
		t.Fatalf("wrong capabilities: %+v", r)
	}
}

type unknownAuxiliary struct{}

func (unknownAuxiliary) Name() string                              { return "unknown" }
func (unknownAuxiliary) Init(*generator.Generator)                 {}
func (unknownAuxiliary) Generate(*generator.FileDescriptor)        {}
func (unknownAuxiliary) GenerateImports(*generator.FileDescriptor) {}
func TestUnknownAuxiliaryDoesNotInheritCapabilities(t *testing.T) {
	r := GeneratePlugin(request(), unknownAuxiliary{}, "aux.go")
	if r.GetSupportedFeatures() != 0 || r.MinimumEdition != nil || r.MaximumEdition != nil {
		t.Fatal("unknown auxiliary inherited capabilities")
	}
}

type editionAuxiliary struct{ unknownAuxiliary }

func (editionAuxiliary) SupportsEditions() bool { return true }
func TestAuxiliaryEditionsOptIn(t *testing.T) {
	req := request()
	req.ProtoFile[0].Syntax = proto.String("editions")
	req.ProtoFile[0].Edition = d.Edition_EDITION_2023.Enum()
	response := GeneratePlugin(req, editionAuxiliary{}, "aux.go")
	if response.GetSupportedFeatures() != uint64(p.CodeGeneratorResponse_FEATURE_SUPPORTS_EDITIONS) {
		t.Fatal(response)
	}
	if response.GetMinimumEdition() != int32(d.Edition_EDITION_2023) || response.GetMaximumEdition() != int32(d.Edition_EDITION_2026) {
		t.Fatal(response)
	}
	response = GeneratePlugin(req, unknownAuxiliary{}, "aux.go")
	if response.GetSupportedFeatures() != 0 || response.MinimumEdition != nil || response.MaximumEdition != nil {
		t.Fatal("unknown plugin inherited Editions")
	}
}

func TestAuxiliaryDoesNotChangeProductionPlugins(t *testing.T) {
	GeneratePlugin(request(), unknownAuxiliary{}, "aux.go")
	req := request()
	req.ProtoFile[0].MessageType = []*d.DescriptorProto{{Name: proto.String("Value")}}
	if err := proto.SetExtension(req.ProtoFile[0].Options, gogoproto.E_EqualAll, proto.Bool(true)); err != nil {
		t.Fatal(err)
	}
	response := Generate(req)
	found := false
	for _, file := range response.File {
		if strings.Contains(file.GetContent(), "Equal(that interface{}) bool") {
			found = true
		}
	}
	if !found {
		t.Fatal("auxiliary plugin leaked into the next production request")
	}
}
