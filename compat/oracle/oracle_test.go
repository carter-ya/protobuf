package oracle

import (
	"bytes"
	"encoding/json"
	pb "github.com/gogo/protobuf/compat/editions/edition2026"
	"github.com/gogo/protobuf/jsonpb"
	gogo "github.com/gogo/protobuf/proto"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"
	"os"
	"reflect"
	"testing"
)

func TestOfficial2026JSON(t *testing.T) {
	raw, err := os.ReadFile(os.Getenv("GOGO_ORACLE_DESCRIPTOR"))
	if err != nil {
		t.Fatal(err)
	}
	set := new(descriptorpb.FileDescriptorSet)
	if err := proto.Unmarshal(raw, set); err != nil {
		t.Fatal(err)
	}
	files, err := protodesc.NewFiles(set)
	if err != nil {
		t.Fatal(err)
	}
	extension, err := files.FindDescriptorByName("pb.enumvalue.json")
	if err != nil {
		t.Fatal(err)
	}
	var types protoregistry.Types
	if err := types.RegisterExtension(dynamicpb.NewExtensionType(extension.(protoreflect.ExtensionDescriptor))); err != nil {
		t.Fatal(err)
	}
	set.Reset()
	if err := (proto.UnmarshalOptions{Resolver: &types}).Unmarshal(raw, set); err != nil {
		t.Fatal(err)
	}
	files, err = protodesc.NewFiles(set)
	if err != nil {
		t.Fatal(err)
	}
	desc, err := files.FindDescriptorByName("compat.editions.v2026.Sample")
	if err != nil {
		t.Fatal(err)
	}
	for _, n := range []pb.Status{0, 1, 2, 7} {
		ours := &pb.Sample{Status: n.Enum(), Statuses: []pb.Status{n}, Values: map[string]pb.Status{"k": n}}
		binary, err := gogo.Marshal(ours)
		if err != nil {
			t.Fatal(err)
		}
		official := dynamicpb.NewMessage(desc.(protoreflect.MessageDescriptor))
		if err := proto.Unmarshal(binary, official); err != nil {
			t.Fatal(err)
		}
		expected, err := (protojson.MarshalOptions{Resolver: &types}).Marshal(official)
		if err != nil {
			t.Fatal(err)
		}
		actual, err := (&jsonpb.Marshaler{}).MarshalToString(ours)
		if err != nil {
			t.Fatal(err)
		}
		var left, right interface{}
		json.Unmarshal(expected, &left)
		json.Unmarshal([]byte(actual), &right)
		if !reflect.DeepEqual(left, right) {
			t.Fatalf("JSON differs for %d: official %s gogo %s", n, expected, actual)
		}
		fromOfficial := new(pb.Sample)
		if err := jsonpb.UnmarshalString(string(expected), fromOfficial); err != nil || !gogo.Equal(ours, fromOfficial) {
			t.Fatalf("official -> gogo: %v", err)
		}
		official.Reset()
		if err := (protojson.UnmarshalOptions{Resolver: &types}).Unmarshal([]byte(actual), official); err != nil {
			t.Fatal(err)
		}
		wire, err := proto.Marshal(official)
		if err != nil {
			t.Fatal(err)
		}
		roundtrip := new(pb.Sample)
		if err := gogo.Unmarshal(wire, roundtrip); err != nil || !gogo.Equal(ours, roundtrip) {
			t.Fatalf("gogo -> official: %x %v", wire, err)
		}
	}
	for _, input := range []string{`{"status":"STATUS_OK"}`, `{"status":"STATUS_ALIAS"}`, `{"status":"\u0053TATUS_OK"}`, `{"status":"ok\"line\n雪"}`, `{"status":""}`} {
		official := dynamicpb.NewMessage(desc.(protoreflect.MessageDescriptor))
		if err := (protojson.UnmarshalOptions{Resolver: &types}).Unmarshal([]byte(input), official); err != nil {
			t.Fatal(err)
		}
		ours := new(pb.Sample)
		if err := jsonpb.UnmarshalString(input, ours); err != nil {
			t.Fatal(err)
		}
		a, _ := proto.Marshal(official)
		b, _ := gogo.Marshal(ours)
		if !bytes.Equal(a, b) {
			t.Fatalf("JSON aliases differ: %s %x %x", input, a, b)
		}
	}
}
