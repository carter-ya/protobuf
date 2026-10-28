package command

import (
	"github.com/gogo/protobuf/proto"
	d "github.com/gogo/protobuf/protoc-gen-gogo/descriptor"
	"github.com/gogo/protobuf/protoc-gen-gogo/generator"
	p "github.com/gogo/protobuf/protoc-gen-gogo/plugin"
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
