package standard

import (
	"github.com/gogo/protobuf/jsonpb"
	"github.com/gogo/protobuf/proto"
	"github.com/gogo/protobuf/types"
	"math"
	"testing"
)

func TestDurationFullRangeAndLegacy(t *testing.T) {
	for _, text := range []string{`"315576000000.999999999s"`, `"-315576000000.999999999s"`, `"0s"`, `"-0.000000001s"`, `"1.010s"`, `"1.000010s"`} {
		var value types.Duration
		if err := UnmarshalString(text, &value); err != nil {
			t.Fatalf("%s: %v", text, err)
		}
		actual, err := (&Marshaler{}).MarshalToString(&value)
		if err != nil || actual != text {
			t.Fatalf("%s: %s %v", text, actual, err)
		}
	}
	for _, text := range []string{`"1m"`, `"1h"`, `"1.0000000001s"`, `"315576000001s"`, `"-315576000001s"`, `"1.s"`, `"+1s"`} {
		if err := UnmarshalString(text, new(types.Duration)); err == nil {
			t.Fatal(text)
		}
	}
	if err := jsonpb.UnmarshalString(`"1m"`, new(types.Duration)); err != nil {
		t.Fatalf("legacy duration changed: %v", err)
	}
	if err := jsonpb.UnmarshalString(`"315576000000s"`, new(types.Duration)); err == nil {
		t.Fatal("legacy duration range unexpectedly changed")
	}
}
func TestTimestampBoundsAndOffsets(t *testing.T) {
	for _, text := range []string{`"0001-01-01T00:00:00Z"`, `"9999-12-31T23:59:59.999999999Z"`, `"1970-01-01T08:00:01+08:00"`} {
		var value types.Timestamp
		if err := UnmarshalString(text, &value); err != nil {
			t.Fatalf("%s: %v", text, err)
		}
		if _, err := (&Marshaler{}).MarshalToString(&value); err != nil {
			t.Fatal(err)
		}
	}
	for _, text := range []string{`"0000-01-01T00:00:00Z"`, `"10000-01-01T00:00:00Z"`, `"1970-01-01T00:00:00.0000000001Z"`, `"1970-01-01T00:00:00,1Z"`, `"1970-01-01T00:00:00+24:00"`} {
		if err := UnmarshalString(text, new(types.Timestamp)); err == nil {
			t.Fatal(text)
		}
	}
	for _, value := range []*types.Timestamp{{Seconds: 253402300800}, {Seconds: -62135596801}, {Nanos: -1}, {Nanos: 1000000000}} {
		if _, err := (&Marshaler{}).MarshalToString(value); err == nil {
			t.Fatal(value)
		}
	}
	if _, err := (&jsonpb.Marshaler{}).MarshalToString(&types.Timestamp{Seconds: 253402300800}); err != nil {
		t.Fatalf("legacy timestamp behavior changed: %v", err)
	}
}
func TestFieldMaskFormats(t *testing.T) {
	value := &types.FieldMask{Paths: []string{"foo_bar", "nested.baz", "*"}}
	text, err := (&Marshaler{}).MarshalToString(value)
	if err != nil || text != `"fooBar,nested.baz,*"` {
		t.Fatalf("%s %v", text, err)
	}
	var roundtrip types.FieldMask
	if err := UnmarshalString(text, &roundtrip); err != nil || !proto.Equal(value, &roundtrip) {
		t.Fatalf("%v %v", roundtrip, err)
	}
	for _, path := range []string{"fooBar", "foo_3_bar", "foo__bar", "foo_", "", "foo,bar"} {
		if _, err := (&Marshaler{}).MarshalToString(&types.FieldMask{Paths: []string{path}}); err == nil {
			t.Fatal(path)
		}
	}
	for _, text := range []string{`"foo_bar"`, `"foo,,bar"`, `null`, `{"paths":[]}`} {
		if err := UnmarshalString(text, new(types.FieldMask)); err == nil {
			t.Fatal(text)
		}
	}
	legacy, err := (&jsonpb.Marshaler{}).MarshalToString(value)
	if err != nil || legacy != `{"paths":["foo_bar","nested.baz","*"]}` {
		t.Fatalf("legacy mask changed: %s %v", legacy, err)
	}
}
func TestAnySpecialAndValueNumbers(t *testing.T) {
	for _, message := range []proto.Message{&types.FieldMask{Paths: []string{"foo_bar"}}, &types.Duration{Seconds: 315576000000}, &types.Empty{}} {
		wire, err := types.MarshalAny(message)
		if err != nil {
			t.Fatal(err)
		}
		text, err := (&Marshaler{}).MarshalToString(wire)
		if err != nil {
			t.Fatal(err)
		}
		var roundtrip types.Any
		if err := UnmarshalString(text, &roundtrip); err != nil || !proto.Equal(wire, &roundtrip) {
			t.Fatalf("%s: %v %v", text, roundtrip, err)
		}
	}
	var empty types.Any
	if err := UnmarshalString(`{}`, &empty); err != nil {
		t.Fatal(err)
	}
	if text, err := (&Marshaler{}).MarshalToString(&empty); err != nil || text != `{}` {
		t.Fatalf("%s %v", text, err)
	}
	for _, text := range []string{`{"@type":"not_a_url","value":"x"}`, `{"@type":"type.googleapis.com/google.protobuf.Duration","value":"1s","extra":1}`} {
		if err := UnmarshalString(text, new(types.Any)); err == nil {
			t.Fatal(text)
		}
	}
	for _, number := range []float64{math.NaN(), math.Inf(1), math.Inf(-1)} {
		if _, err := (&Marshaler{}).MarshalToString(&types.Value{Kind: &types.Value_NumberValue{NumberValue: number}}); err == nil {
			t.Fatal("non-finite Value accepted")
		}
	}
}
