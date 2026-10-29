package generator

import (
	"fmt"
	"github.com/gogo/protobuf/proto"
	d "github.com/gogo/protobuf/protoc-gen-gogo/descriptor"
)

// Minimal wire view of the pinned official go_features.proto. Keeping this in
// the generator avoids adding a modern runtime dependency to old consumers.
// Gogo keeps its historical Open Struct API, including for Edition 2024 files.
// Explicit requests for different output APIs must fail, not be ignored.
type goFeatures struct {
	LegacyJSON       *bool  `protobuf:"varint,1,opt,name=legacy_unmarshal_json_enum"`
	API              *int32 `protobuf:"varint,2,opt,name=api_level"`
	Strip            *int32 `protobuf:"varint,3,opt,name=strip_enum_prefix"`
	Optimize         *int32 `protobuf:"varint,4,opt,name=optimize_mode"`
	XXX_unrecognized []byte
}

func (m *goFeatures) Reset()         { *m = goFeatures{} }
func (m *goFeatures) String() string { return proto.CompactTextString(m) }
func (*goFeatures) ProtoMessage()    {}

var goFeatureExtension = &proto.ExtensionDesc{ExtendedType: (*d.FeatureSet)(nil), ExtensionType: (*goFeatures)(nil), Field: 1002, Name: "pb.go", Tag: "bytes,1002,opt,name=go"}

func validateGoFeatures(features *d.FeatureSet) error {
	value, err := proto.GetExtension(features, goFeatureExtension)
	if err != nil {
		return fmt.Errorf("cannot decode Go features: %v", err)
	}
	f, ok := value.(*goFeatures)
	if !ok {
		return fmt.Errorf("unexpected Go features type")
	}
	if f.API != nil && *f.API != 0 && *f.API != 1 {
		return fmt.Errorf("Go api_level %d is unsupported; gogo preserves the Open Struct API", *f.API)
	}
	if f.Strip != nil && *f.Strip != 1 {
		return fmt.Errorf("Go strip_enum_prefix %d is unsupported; gogo preserves enum prefixes", *f.Strip)
	}
	if f.LegacyJSON != nil || f.Optimize != nil || len(f.XXX_unrecognized) > 0 {
		return fmt.Errorf("explicit Go feature is not supported by the gogo backend")
	}
	return nil
}
