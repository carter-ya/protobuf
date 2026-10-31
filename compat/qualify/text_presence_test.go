package qualify

import (
	"testing"

	legacy "github.com/gogo/protobuf/proto"
	standard "github.com/gogo/protobuf/textpb/standard"
	gogotypes "github.com/gogo/protobuf/types"
	"google.golang.org/protobuf/encoding/prototext"
	modern "google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/known/structpb"
)

type namePart struct {
	NamePart    *string `protobuf:"bytes,1,req,name=name_part,json=namePart"`
	IsExtension *bool   `protobuf:"varint,2,req,name=is_extension,json=isExtension"`
}

func (m *namePart) Reset()         { *m = namePart{} }
func (m *namePart) String() string { return legacy.CompactTextString(m) }
func (*namePart) ProtoMessage()    {}
func (*namePart) Descriptor() ([]byte, []int) {
	return new(descriptorpb.UninterpretedOption_NamePart).Descriptor()
}
func TestStandardTextRequiredOracle(t *testing.T) {
	for _, complete := range []bool{false, true} {
		sdk, official := new(namePart), new(descriptorpb.UninterpretedOption_NamePart)
		if complete {
			sdk.NamePart = legacy.String("")
			sdk.IsExtension = legacy.Bool(false)
			official.NamePart = modern.String("")
			official.IsExtension = modern.Bool(false)
		}
		output, sdkErr := (&standard.Marshaler{}).MarshalToString(sdk)
		_, officialErr := prototext.Marshal(official)
		if (sdkErr == nil) != (officialErr == nil) {
			t.Fatalf("required policy differs: SDK %v official %v", sdkErr, officialErr)
		}
		if sdkErr != nil {
			continue
		}
		var back descriptorpb.UninterpretedOption_NamePart
		if err := prototext.Unmarshal([]byte(output), &back); err != nil || !modern.Equal(official, &back) {
			t.Fatalf("required zero values lost in %q: %v", output, err)
		}
	}
}
func TestStandardTextNilMessageOracle(t *testing.T) {
	for _, test := range []struct {
		sdk      legacy.Message
		official modern.Message
	}{
		{&gogotypes.Value{Kind: (*gogotypes.Value_StringValue)(nil)}, &structpb.Value{Kind: (*structpb.Value_StringValue)(nil)}},
		{&gogotypes.Value{Kind: &gogotypes.Value_StructValue{}}, &structpb.Value{Kind: &structpb.Value_StructValue{}}},
		{&gogotypes.ListValue{Values: []*gogotypes.Value{nil}}, &structpb.ListValue{Values: []*structpb.Value{nil}}},
		{&gogotypes.Struct{Fields: map[string]*gogotypes.Value{"a": nil}}, &structpb.Struct{Fields: map[string]*structpb.Value{"a": nil}}},
	} {
		for _, compact := range []bool{false, true} {
			output, err := (&standard.Marshaler{Compact: compact}).MarshalToString(test.sdk)
			if err != nil {
				t.Fatalf("%T: %v", test.sdk, err)
			}
			reference, err := prototext.Marshal(test.official)
			if err != nil {
				t.Fatal(err)
			}
			got, want := test.official.ProtoReflect().Type().New().Interface(), test.official.ProtoReflect().Type().New().Interface()
			if err := prototext.Unmarshal([]byte(output), got); err != nil {
				t.Fatalf("official rejects SDK output %q: %v", output, err)
			}
			if err := prototext.Unmarshal(reference, want); err != nil {
				t.Fatal(err)
			}
			if !modern.Equal(got, want) {
				t.Fatalf("nil presence differs: SDK %s official %s", got, want)
			}
		}
	}
}
