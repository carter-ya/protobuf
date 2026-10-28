package generator

import (
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
		if err := proto.SetExtension(features, goFeatureExtension, &goFeatures{API: proto.Int32(api)}); err != nil {
			t.Fatal(err)
		}
		err := validateGoFeatures(features)
		if (api <= 1) != (err == nil) {
			t.Fatalf("api %d: %v", api, err)
		}
	}
}
