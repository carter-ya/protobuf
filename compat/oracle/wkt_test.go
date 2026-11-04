package oracle

import (
	"bytes"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/gogo/protobuf/jsonpb/standard"
	gogo "github.com/gogo/protobuf/proto"
	old "github.com/gogo/protobuf/types"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/apipb"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/typepb"
)

func TestStandardWKTInteroperability(t *testing.T) {
	mask := &old.FieldMask{Paths: []string{"foo_bar", "nested.baz"}}
	packed, err := old.MarshalAny(mask)
	if err != nil {
		t.Fatal(err)
	}
	for _, pair := range []struct {
		ours     gogo.Message
		official proto.Message
	}{
		{&old.Duration{Seconds: 315576000000, Nanos: 999999999}, new(durationpb.Duration)},
		{&old.Duration{Seconds: -315576000000, Nanos: -999999999}, new(durationpb.Duration)},
		{&old.Timestamp{Seconds: -62135596800}, new(timestamppb.Timestamp)},
		{&old.Timestamp{Seconds: 253402300799, Nanos: 999999999}, new(timestamppb.Timestamp)},
		{&old.Type{Name: "M", Syntax: old.Syntax_SYNTAX_EDITIONS, Edition: "2026"}, new(typepb.Type)},
		{&old.Enum{Name: "E", Syntax: old.Syntax_SYNTAX_EDITIONS, Edition: "2024"}, new(typepb.Enum)},
		{&old.Api{Name: "A", Syntax: old.Syntax_SYNTAX_EDITIONS, Edition: "2023"}, new(apipb.Api)},
		{&old.Method{Name: "Call", Syntax: old.Syntax_SYNTAX_EDITIONS, Edition: "2023"}, new(apipb.Method)},
		{mask, new(fieldmaskpb.FieldMask)}, {packed, new(anypb.Any)},
		{new(old.Any), new(anypb.Any)}, {new(old.Empty), new(emptypb.Empty)},
	} {
		wire, err := gogo.Marshal(pair.ours)
		if err != nil {
			t.Fatal(err)
		}
		if err := proto.Unmarshal(wire, pair.official); err != nil {
			t.Fatal(err)
		}
		expected, err := protojson.Marshal(pair.official)
		if err != nil {
			t.Fatal(err)
		}
		actual, err := (&standard.Marshaler{}).MarshalToString(pair.ours)
		if err != nil {
			t.Fatal(err)
		}
		var a, b interface{}
		json.Unmarshal(expected, &a)
		json.Unmarshal([]byte(actual), &b)
		if !reflect.DeepEqual(a, b) {
			t.Fatalf("%T: official %s, gogo %s", pair.ours, expected, actual)
		}
		roundtrip := reflect.New(reflect.TypeOf(pair.ours).Elem()).Interface().(gogo.Message)
		if err := standard.UnmarshalString(string(expected), roundtrip); err != nil || !gogo.Equal(pair.ours, roundtrip) {
			t.Fatalf("official -> gogo %T: %v", pair.ours, err)
		}
		if err := protojson.Unmarshal([]byte(actual), pair.official); err != nil {
			t.Fatalf("gogo -> official %T: %v", pair.ours, err)
		}
		actualWire, err := proto.Marshal(pair.official)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(wire, actualWire) {
			t.Fatalf("wire changed for %T", pair.ours)
		}
	}
}
