package standard

import (
	"strings"
	"testing"

	"github.com/gogo/protobuf/jsonpb"
	pb "github.com/gogo/protobuf/jsonpb/jsonpb_test_proto"
	"github.com/gogo/protobuf/proto"
	"github.com/gogo/protobuf/types"
)

// Keep the historical unkeyed configuration literals source-compatible.
var _ = jsonpb.Marshaler{false, false, "", false, nil}
var _ = jsonpb.Unmarshaler{false, nil}

func TestCompleteDocumentAndUnicode(t *testing.T) {
	for _, input := range []string{`{"oInt32":1} {}`, `{"oInt32":1} garbage`, `{"oString":"\ud800"}`, `{"oString":"\udc00"}`, "{\"oString\":\"\xff\"}", `{"oInt32":1,"oInt32":2}`, `{"oInt32":1,"o_int32":2}`, `{"oInt32":" 1"}`} {
		if err := UnmarshalString(input, new(pb.Simple)); err == nil {
			t.Fatalf("accepted %q", input)
		}
	}
	var value pb.Simple
	if err := UnmarshalString(`{"oString":"\ud83d\ude00"}`, &value); err != nil || value.GetOString() != "😀" {
		t.Fatalf("surrogate pair: %v %v", value, err)
	}
	value.OInt32 = proto.Int32(9)
	if err := UnmarshalString(`{}`, &value); err != nil || value.OInt32 != nil {
		t.Fatalf("destination not reset: %v %v", value, err)
	}
	if err := jsonpb.UnmarshalString(`{"oInt32":" 1"}`, &value); err != nil {
		t.Fatalf("legacy whitespace behavior changed: %v", err)
	}
}

func TestPresenceAndDefaults(t *testing.T) {
	value := &pb.Simple{OInt32: proto.Int32(0)}
	text, err := (&Marshaler{EmitDefaults: true}).MarshalToString(value)
	if err != nil || !strings.Contains(text, `"oInt32":0`) || strings.Contains(text, `"oString"`) {
		t.Fatalf("presence: %s %v", text, err)
	}
	legacy, err := (&jsonpb.Marshaler{EmitDefaults: true}).MarshalToString(value)
	if err != nil || !strings.Contains(legacy, `"oString":null`) {
		t.Fatalf("legacy defaults changed: %s %v", legacy, err)
	}
	if _, err := (&Marshaler{}).MarshalToString(&types.StringValue{Value: string([]byte{255})}); err == nil {
		t.Fatal("invalid UTF-8 emitted")
	}
}

func TestNullCollections(t *testing.T) {
	// The official proto3 fixture already covers primitive, message and oneof fields.
	for _, input := range []string{`{"repeatedString":[null]}`, `{"repeatedInt32":[null]}`} {
		// Use a tagged message with ordinary repeated representations below.
		if err := UnmarshalString(input, new(nullProbe)); err == nil {
			t.Fatalf("accepted %s", input)
		}
	}
	var value nullProbe
	if err := UnmarshalString(`{"values":[null,1,"x"]}`, &value); err != nil || len(value.Values) != 3 || value.Values[0].GetNullValue() != types.NullValue_NULL_VALUE {
		t.Fatalf("Value null: %v %v", value, err)
	}
	if err := UnmarshalString(`{"repeatedString":null}`, &value); err != nil {
		t.Fatal(err)
	}
}

type nullProbe struct {
	Strings []string       `protobuf:"bytes,1,rep,name=repeated_string,json=repeatedString,proto3" json:"repeatedString,omitempty"`
	Numbers []int32        `protobuf:"varint,2,rep,name=repeated_int32,json=repeatedInt32,proto3" json:"repeatedInt32,omitempty"`
	Values  []*types.Value `protobuf:"bytes,3,rep,name=values,proto3" json:"values,omitempty"`
}

func (m *nullProbe) Reset()         { *m = nullProbe{} }
func (m *nullProbe) String() string { return proto.CompactTextString(m) }
func (*nullProbe) ProtoMessage()    {}

func TestNilDestination(t *testing.T) {
	var value *pb.Simple
	if err := UnmarshalString(`{}`, value); err == nil {
		t.Fatal("nil destination accepted")
	}
}

func TestOneofNullAndConflicts(t *testing.T) {
	for i := 0; i < 100; i++ {
		for _, input := range []string{`{"title":"x","salary":null}`, `{"salary":null,"title":"x"}`} {
			var value pb.MsgWithOneof
			if err := UnmarshalString(input, &value); err != nil || value.GetTitle() != "x" {
				t.Fatalf("%s: %v %v", input, value, err)
			}
		}
	}
	if err := UnmarshalString(`{"title":"x","salary":1}`, new(pb.MsgWithOneof)); err == nil {
		t.Fatal("oneof conflict accepted")
	}
}
