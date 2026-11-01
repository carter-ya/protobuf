package generator

import (
	gofeaturespb "github.com/gogo/protobuf/gofeatures"
	"github.com/gogo/protobuf/proto"
	d "github.com/gogo/protobuf/protoc-gen-gogo/descriptor"
	"strings"
	"testing"
)

func TestEditionBounds(t *testing.T) {
	for _, n := range []d.Edition{0, 999, 1003, 2147483647} {
		_, _, err := prepareEdition(&d.FileDescriptorProto{Name: proto.String("future.proto"), Syntax: proto.String("editions"), Edition: n.Enum()}, nil)
		if err == nil || !strings.Contains(err.Error(), "unsupported edition") {
			t.Fatalf("unsupported edition %d accepted: %v", n, err)
		}
	}
}
func TestGoFeaturePolicy(t *testing.T) {
	for _, api := range []int32{0, 1, 2, 3, 99} {
		features := &d.FeatureSet{}
		if err := proto.SetExtension(features, goFeatureExtension, &goFeatures{ApiLevel: gofeaturespb.GoFeatures_APILevel(api).Enum()}); err != nil {
			t.Fatal(err)
		}
		err := validateGoFeatures(features)
		if (api <= 2) != (err == nil) {
			t.Fatalf("api %d: %v", api, err)
		}
	}
}

func TestGoFeatureUnknownFields(t *testing.T) {
	var value goFeatures
	if err := proto.Unmarshal([]byte{0x98, 0x06, 1}, &value); err != nil {
		t.Fatal(err)
	}
	features := &d.FeatureSet{}
	if err := proto.SetExtension(features, goFeatureExtension, &value); err != nil {
		t.Fatal(err)
	}
	if err := validateGoFeatures(features); err == nil {
		t.Fatal("unrecognized Go feature was silently accepted")
	}
}
