package gogoreflect

import (
	"bytes"
	"compress/gzip"
	"testing"

	_ "github.com/gogo/protobuf/gogoproto"
	v2 "google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
)

func compressedFile(t *testing.T, f *descriptorpb.FileDescriptorProto) []byte {
	t.Helper()
	raw, e := v2.Marshal(f)
	if e != nil {
		t.Fatal(e)
	}
	var b bytes.Buffer
	z := gzip.NewWriter(&b)
	z.Write(raw)
	z.Close()
	return b.Bytes()
}
func TestHistoricalDescriptorAliasCoexistence(t *testing.T) {
	u := &schemaUniverse{files: new(protoregistry.Files), types: new(protoregistry.Types), raw: make(map[string][]byte)}
	first := &descriptorpb.FileDescriptorProto{Name: v2.String("first.proto"), Package: v2.String("alias.first"), Syntax: v2.String("proto3"), Dependency: []string{"gogoproto/gogo.proto"}, MessageType: []*descriptorpb.DescriptorProto{{Name: v2.String("First")}}}
	second := &descriptorpb.FileDescriptorProto{Name: v2.String("second.proto"), Package: v2.String("alias.second"), Syntax: v2.String("proto3"), Dependency: []string{"github.com/gogo/protobuf/gogoproto/gogo.proto"}, MessageType: []*descriptorpb.DescriptorProto{{Name: v2.String("Second")}}}
	a, e := u.load(compressedFile(t, first), "", make(map[string]bool))
	if e != nil {
		t.Fatal(e)
	}
	b, e := u.load(compressedFile(t, second), "", make(map[string]bool))
	if e != nil {
		t.Fatal(e)
	}
	if a.Imports().Get(0).FileDescriptor != b.Imports().Get(0).FileDescriptor {
		t.Fatal("aliases created different descriptor identities")
	}
	if a.Imports().Get(0).Path() != "gogoproto/gogo.proto" {
		t.Fatal("unexpected canonical dependency", a.Imports().Get(0).Path())
	}
}
