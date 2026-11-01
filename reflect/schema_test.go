package gogoreflect

import (
	"bytes"
	"compress/gzip"
	"fmt"
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

// Native package registration has finished, but another type may still discover
// option extensions on its first reflection. Local registries must remain safe.
func TestColdMetadataResolverRace(t *testing.T) {
	r := new(Registry)
	done := make(chan struct{})
	reader := make(chan struct{})
	go func() {
		defer close(reader)
		for {
			select {
			case <-done:
				return
			default:
			}
			r.FindExtensionByName("coldmetadata.option_0")
			r.FindExtensionByNumber("google.protobuf.FileOptions", 80000)
		}
	}()
	for i := 0; i < 40; i++ {
		file := &descriptorpb.FileDescriptorProto{Name: v2.String(fmt.Sprintf("coldmetadata_%d.proto", i)), Package: v2.String("coldmetadata"), Syntax: v2.String("proto2"), Dependency: []string{"google/protobuf/descriptor.proto"}, Extension: []*descriptorpb.FieldDescriptorProto{{Name: v2.String(fmt.Sprintf("option_%d", i)), Number: v2.Int32(int32(80000 + i)), Extendee: v2.String(".google.protobuf.FileOptions"), Type: descriptorpb.FieldDescriptorProto_TYPE_INT32.Enum(), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum()}}}
		raw := compressedFile(t, file)
		schemas.Lock()
		_, err := schemas.load(raw, "", make(map[string]bool))
		schemas.Unlock()
		if err != nil {
			close(done)
			<-reader
			t.Fatal(err)
		}
	}
	close(done)
	<-reader
}
