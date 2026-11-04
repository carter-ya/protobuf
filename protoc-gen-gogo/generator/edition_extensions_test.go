package generator

import (
	"strings"
	"testing"

	"github.com/gogo/protobuf/proto"
	d "github.com/gogo/protobuf/protoc-gen-gogo/descriptor"
	p "github.com/gogo/protobuf/protoc-gen-gogo/plugin"
)

func TestEditionExtensionDependencyRuntimeGuard(t *testing.T) {
	dependency := &d.FileDescriptorProto{
		Name: proto.String("dependency.proto"), Package: proto.String("dependency"),
		Syntax: proto.String("editions"), Edition: d.Edition_EDITION_2023.Enum(),
		Options:     &d.FileOptions{GoPackage: proto.String("example.com/dependency")},
		MessageType: []*d.DescriptorProto{{Name: proto.String("Extensible"), ExtensionRange: []*d.DescriptorProto_ExtensionRange{{Start: proto.Int32(100), End: proto.Int32(200)}}}},
	}
	parent := &d.FileDescriptorProto{
		Name: proto.String("parent.proto"), Package: proto.String("parent"),
		Syntax: proto.String("editions"), Edition: d.Edition_EDITION_2023.Enum(),
		Dependency: []string{"dependency.proto"}, Options: &d.FileOptions{GoPackage: proto.String("example.com/parent")},
		MessageType: []*d.DescriptorProto{{Name: proto.String("Parent"), Field: []*d.FieldDescriptorProto{{
			Name: proto.String("child"), Number: proto.Int32(1), Label: d.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
			Type: d.FieldDescriptorProto_TYPE_MESSAGE.Enum(), TypeName: proto.String(".dependency.Extensible"),
		}}}},
	}
	g := New()
	g.Request = &p.CodeGeneratorRequest{FileToGenerate: []string{"parent.proto"}, ProtoFile: []*d.FileDescriptorProto{dependency, parent}}
	g.CommandLineParameters("")
	g.WrapTypes()
	g.SetPackageNames()
	g.BuildTypeNameMap()
	g.GenerateAllFiles()
	if len(g.Response.File) != 1 || !strings.Contains(g.Response.File[0].GetContent(), "GoGoProtoPackageIsVersion13") {
		t.Fatal("an enclosing message without its own extension range must require the deferred extension runtime")
	}
}
