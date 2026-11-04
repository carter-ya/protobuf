package editions_test

import (
	"bytes"
	"compress/gzip"
	pb "github.com/gogo/protobuf/compat/editions/edition2024"
	"github.com/gogo/protobuf/jsonpb"
	"github.com/gogo/protobuf/proto"
	d "github.com/gogo/protobuf/protoc-gen-gogo/descriptor"
	"io/ioutil"
	"strings"
	"testing"
)

func TestEdition2024OptionsAndPresence(t *testing.T) {
	m := &pb.Sample{Value: proto.Int32(0), Status: pb.Status_STATUS_OK.Enum()}
	b := marshal(t, m)
	if !bytes.Equal(b, wire("08001001")) {
		t.Fatalf("2024 presence: %x", b)
	}
	j, err := (&jsonpb.Marshaler{}).MarshalToString(m)
	if err != nil || j != `{"value":0,"status":"STATUS_OK"}` {
		t.Fatalf("JSON: %s %v", j, err)
	}
	raw, _ := m.Descriptor()
	z, err := gzip.NewReader(bytes.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := ioutil.ReadAll(z)
	z.Close()
	if err != nil {
		t.Fatal(err)
	}
	file := new(d.FileDescriptorProto)
	if err := proto.Unmarshal(decoded, file); err != nil {
		t.Fatal(err)
	}
	if file.GetEdition() != d.Edition_EDITION_2024 || len(file.OptionDependency) != 2 {
		t.Fatalf("option imports lost: %+v", file.OptionDependency)
	}
	options := file.MessageType[0].Field[0].Options
	runtime := &proto.ExtensionDesc{ExtendedType: (*d.FieldOptions)(nil), ExtensionType: (*string)(nil), Field: 50001, Name: "compat.editions.options.runtime_note", Tag: "bytes,50001,opt,name=runtime_note"}
	source := &proto.ExtensionDesc{ExtendedType: (*d.FieldOptions)(nil), ExtensionType: (*string)(nil), Field: 50002, Name: "compat.editions.options.source_note", Tag: "bytes,50002,opt,name=source_note"}
	value, err := proto.GetExtension(options, runtime)
	if err != nil || *value.(*string) != "keep" {
		t.Fatalf("runtime option lost: %v %v", value, err)
	}
	if proto.HasExtension(options, source) {
		t.Fatal("source-retained option leaked into runtime descriptor")
	}
	// Both are option-only dependencies and have no generated Go packages here.
	for _, dep := range file.Dependency {
		if strings.Contains(dep, "options.proto") || strings.Contains(dep, "go_features.proto") {
			t.Fatal("option dependency became a runtime dependency")
		}
	}
}
