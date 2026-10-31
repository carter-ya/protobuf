package generator

import (
	"fmt"
	gofeaturespb "github.com/gogo/protobuf/gofeatures"
	"github.com/gogo/protobuf/proto"
	d "github.com/gogo/protobuf/protoc-gen-gogo/descriptor"
)

// Use the Go 1.15-compatible binding for the pinned Go feature extension. A
// single descriptor also avoids conflicts between decoding and re-decoding.
// Different output APIs must still fail until their implementation is qualified.
type goFeatures = gofeaturespb.GoFeatures

var goFeatureExtension = gofeaturespb.E_Go

func readGoFeatures(features *d.FeatureSet) (*goFeatures, error) {
	value, err := proto.GetExtension(features, goFeatureExtension)
	if err != nil {
		return nil, fmt.Errorf("cannot decode Go features: %v", err)
	}
	f, ok := value.(*goFeatures)
	if !ok || f == nil {
		return nil, fmt.Errorf("unexpected Go features type")
	}
	return f, nil
}

func validateGoFeatures(features *d.FeatureSet) error {
	f, err := readGoFeatures(features)
	if err != nil {
		return err
	}
	if f.ApiLevel != nil && *f.ApiLevel != 0 && *f.ApiLevel != 1 {
		return fmt.Errorf("Go api_level %d is unsupported; gogo preserves the Open Struct API", *f.ApiLevel)
	}
	if f.StripEnumPrefix != nil && (*f.StripEnumPrefix < 0 || *f.StripEnumPrefix > 3) {
		return fmt.Errorf("Go strip_enum_prefix %d is invalid", *f.StripEnumPrefix)
	}
	known := proto.Clone(f)
	proto.DiscardUnknown(known)
	if f.OptimizeMode != nil && (*f.OptimizeMode < 0 || *f.OptimizeMode > 2) {
		return fmt.Errorf("Go optimize_mode %d is invalid", *f.OptimizeMode)
	}
	if !proto.Equal(f, known) {
		return fmt.Errorf("explicit Go feature is not supported by the gogo backend")
	}
	return nil
}

// Editions inherit false from the official proto3 legacy-JSON default. File
// values propagate to nested enums; an enum declaration can override the file.
func enumLegacyJSON(features *d.FeatureSet, inherited bool) (bool, error) {
	if features == nil || !proto.HasExtension(features, goFeatureExtension) {
		return inherited, nil
	}
	value, err := proto.GetExtension(features, goFeatureExtension)
	if err != nil {
		return false, fmt.Errorf("cannot decode Go features: %v", err)
	}
	f, ok := value.(*goFeatures)
	if !ok || f == nil {
		return false, fmt.Errorf("unexpected Go features type")
	}
	if f.LegacyUnmarshalJsonEnum != nil {
		return *f.LegacyUnmarshalJsonEnum, nil
	}
	return inherited, nil
}
