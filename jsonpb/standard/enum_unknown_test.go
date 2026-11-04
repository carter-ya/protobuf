package standard

import (
	"reflect"
	"strings"
	"testing"

	"github.com/gogo/protobuf/jsonpb"
	pb "github.com/gogo/protobuf/jsonpb/jsonpb_test_proto"
	"github.com/gogo/protobuf/proto"
	"github.com/gogo/protobuf/types"
)

type enumIgnoreProbe struct {
	Color   *pb.Widget_Color           `protobuf:"varint,1,opt,name=color,enum=jsonpb.Widget_Color"`
	Colors  []pb.Widget_Color          `protobuf:"varint,2,rep,name=colors,enum=jsonpb.Widget_Color"`
	Mapping map[string]pb.Widget_Color `protobuf:"bytes,3,rep,name=mapping" protobuf_key:"bytes,1,opt,name=key" protobuf_val:"varint,2,opt,name=value,enum=jsonpb.Widget_Color"`
	Choice  isEnumIgnoreProbe_Choice   `protobuf_oneof:"choice"`
}

func (m *enumIgnoreProbe) Reset()         { *m = enumIgnoreProbe{} }
func (m *enumIgnoreProbe) String() string { return proto.CompactTextString(m) }
func (*enumIgnoreProbe) ProtoMessage()    {}
func (*enumIgnoreProbe) XXX_OneofWrappers() []interface{} {
	return []interface{}{(*enumIgnoreSelected)(nil), (*enumIgnoreNumber)(nil)}
}

type isEnumIgnoreProbe_Choice interface{ isEnumIgnoreProbe_Choice() }
type enumIgnoreSelected struct {
	Selected pb.Widget_Color `protobuf:"varint,4,opt,name=selected,enum=jsonpb.Widget_Color,oneof"`
}

func (*enumIgnoreSelected) isEnumIgnoreProbe_Choice() {}

type enumIgnoreNumber struct {
	Number int32 `protobuf:"varint,5,opt,name=number,oneof"`
}

func (*enumIgnoreNumber) isEnumIgnoreProbe_Choice() {}

func TestIgnoreUnknownEnumNames(t *testing.T) {
	input := `{"color":"UNKNOWN","colors":["RED","UNKNOWN","GREEN"],"mapping":{"keep":"BLUE","drop":"UNKNOWN"}}`
	var value enumIgnoreProbe
	if err := (&Unmarshaler{AllowUnknownFields: true}).Unmarshal(strings.NewReader(input), &value); err != nil {
		t.Fatal(err)
	}
	if value.Color != nil || !reflect.DeepEqual(value.Colors, []pb.Widget_Color{pb.Widget_RED, pb.Widget_GREEN}) || !reflect.DeepEqual(value.Mapping, map[string]pb.Widget_Color{"keep": pb.Widget_BLUE}) {
		t.Fatalf("ignored enum invented values: %+v", value)
	}
	if err := UnmarshalString(input, new(enumIgnoreProbe)); err == nil {
		t.Fatal("strict mode ignored enum")
	}
	if err := (&jsonpb.Unmarshaler{AllowUnknownFields: true}).Unmarshal(strings.NewReader(input), new(enumIgnoreProbe)); err == nil {
		t.Fatal("legacy behavior changed")
	}
	for i := 0; i < 50; i++ {
		if err := (&Unmarshaler{AllowUnknownFields: true}).Unmarshal(strings.NewReader(`{"selected":"UNKNOWN","number":7}`), &value); err != nil {
			t.Fatal(err)
		}
		if number, ok := value.Choice.(*enumIgnoreNumber); !ok || number.Number != 7 {
			t.Fatal("unknown enum consumed oneof presence")
		}
	}
	if err := (&Unmarshaler{AllowUnknownFields: true}).Unmarshal(strings.NewReader(`{"selected":"UNKNOWN"}`), &value); err != nil || value.Choice != nil {
		t.Fatalf("ignored oneof invented presence: %v", err)
	}
	for _, input := range []string{`{"colors":[true]}`, `{"color":{}}`, `{"mapping":{"x":[]}}`, `{"colors":"UNKNOWN"}`, `{"colors":"RED"}`, `{"colors":[["UNKNOWN"]]}`} {
		if err := (&Unmarshaler{AllowUnknownFields: true}).Unmarshal(strings.NewReader(input), &value); err == nil {
			t.Fatalf("ignored malformed enum input: %s", input)
		}
	}
	if err := (&Unmarshaler{AllowUnknownFields: true}).Unmarshal(strings.NewReader(`{"selected":"RED","number":7}`), &value); err == nil {
		t.Fatal("real oneof conflict accepted")
	}
	if err := (&Unmarshaler{AllowUnknownFields: true}).Unmarshal(strings.NewReader(`{"colors":["UNKNOWN"],"mapping":{"x":"UNKNOWN"}}`), &value); err != nil || len(value.Colors) != 0 || len(value.Mapping) != 0 {
		t.Fatalf("all-unknown collections: %+v %v", value, err)
	}
}

func TestStandardJSONNestingBoundary(t *testing.T) {
	for _, depth := range []int{99, 100, 101, 200} {
		input := strings.Repeat("[", depth) + "0" + strings.Repeat("]", depth)
		err := UnmarshalString(input, new(types.Value))
		if depth <= 100 && err != nil || depth > 100 && err == nil {
			t.Fatalf("depth %d: %v", depth, err)
		}
		legacy := new(types.Value)
		if err := jsonpb.UnmarshalString(input, legacy); err != nil {
			t.Fatalf("legacy depth %d changed: %v", depth, err)
		}
		_, err = (&Marshaler{}).MarshalToString(legacy)
		if depth <= 100 && err != nil || depth > 100 && err == nil {
			t.Fatalf("marshal depth %d: %v", depth, err)
		}
	}
}
