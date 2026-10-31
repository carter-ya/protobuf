package standard

import (
	"math"
	"strings"
	"testing"

	pb "github.com/gogo/protobuf/jsonpb/jsonpb_test_proto"
	"github.com/gogo/protobuf/proto"
)

var _ = proto.TextMarshaler{false, false} // historical configuration layout

type probe struct {
	XXX_unrecognized []byte
	Float            float32          `protobuf:"fixed32,1,opt,name=float,proto3"`
	Double           *float64         `protobuf:"fixed64,2,opt,name=double"`
	Checked          string           `protobuf:"bytes,3,opt,name=checked,proto3"`
	Unchecked        string           `protobuf:"bytes,4,opt,name=unchecked,proto3,utf8_none"`
	Closed           *pb.Widget_Color `protobuf:"varint,5,opt,name=closed,enum=jsonpb.Widget_Color"`
	Open             pb.Widget_Color  `protobuf:"varint,6,opt,name=open,proto3,enum=jsonpb.Widget_Color"`
	Bytes            []byte           `protobuf:"bytes,7,opt,name=bytes,proto3"`
}

func (m *probe) Reset()         { *m = probe{} }
func (m *probe) String() string { return proto.CompactTextString(m) }
func (*probe) ProtoMessage()    {}

func TestStandardFloatSyntax(t *testing.T) {
	for _, tc := range []struct {
		input string
		want  float64
	}{
		{"1F", 1}, {"-.123F", -.123}, {"1.5f", 1.5}, {".125e2F", 12.5},
		{"1e9999", math.Inf(1)}, {"-1e9999", math.Inf(-1)},
		{"1e18446744073709551616", math.Inf(1)}, {"-1e-18446744073709551616", math.Copysign(0, -1)},
		{"INF", math.Inf(1)}, {"-inFINITY", math.Inf(-1)}, {"nAn", math.NaN()},
	} {
		var value probe
		if err := UnmarshalString("double: "+tc.input, &value); err != nil {
			t.Fatalf("%s: %v", tc.input, err)
		}
		if value.Double == nil {
			t.Fatal("double lost presence")
		}
		got := *value.Double
		if math.IsNaN(tc.want) {
			if !math.IsNaN(got) {
				t.Fatal("NaN lost")
			}
		} else if got != tc.want || math.Signbit(got) != math.Signbit(tc.want) {
			t.Fatalf("%s: %v", tc.input, got)
		}
	}
	for _, input := range []string{"012", "-012", "0x1p1", "0x1", "+1", "00.1", "nanF", "infF", "1e", "."} {
		if err := UnmarshalString("float: "+input, new(probe)); err == nil {
			t.Fatalf("accepted %s", input)
		}
	}
	var value probe
	if err := UnmarshalString("double: - # separated sign\n 2.5F", &value); err != nil || value.Double == nil || *value.Double != -2.5 {
		t.Fatalf("separated sign: %v", err)
	}
	// Existing text callers keep their historical float grammar and tolerance.
	if err := proto.UnmarshalText("float: 012", &value); err != nil || value.Float != 12 {
		t.Fatalf("legacy octal-looking float changed: %v", err)
	}
	if err := proto.UnmarshalText("float: 1F", &value); err == nil {
		t.Fatal("legacy uppercase suffix behavior changed")
	}
}

func TestStandardNegativeZeroAndUTF8(t *testing.T) {
	value := &probe{Float: float32(math.Copysign(0, -1))}
	for _, compact := range []bool{false, true} {
		text, err := (&Marshaler{Compact: compact}).MarshalToString(value)
		if err != nil || !strings.Contains(text, "-0") {
			t.Fatalf("negative zero %q %v", text, err)
		}
		var parsed probe
		if err := UnmarshalString(text, &parsed); err != nil || !math.Signbit(float64(parsed.Float)) {
			t.Fatalf("zero sign lost: %v", err)
		}
	}
	if text := proto.MarshalTextString(value); text != "" {
		t.Fatalf("legacy zero output changed: %q", text)
	}
	if err := UnmarshalString(`checked: "\xff"`, new(probe)); err == nil {
		t.Fatal("accepted invalid checked UTF-8")
	}
	if _, err := (&Marshaler{}).MarshalToString(&probe{Checked: string([]byte{255})}); err == nil {
		t.Fatal("emitted invalid checked UTF-8")
	}
	var parsed probe
	if err := UnmarshalString(`unchecked: "\xff"`, &parsed); err != nil || parsed.Unchecked != string([]byte{255}) {
		t.Fatalf("NONE rejected: %v", err)
	}
	if err := proto.UnmarshalText(`checked: "\xff"`, &parsed); err != nil {
		t.Fatalf("legacy UTF-8 tolerance changed: %v", err)
	}
}

func TestStandardEnumPolicyAndNil(t *testing.T) {
	if err := UnmarshalString("closed: 7", new(probe)); err == nil {
		t.Fatal("unknown closed enum accepted")
	}
	if err := UnmarshalString("open: 7", new(probe)); err != nil {
		t.Fatal(err)
	}
	if err := proto.UnmarshalText("closed: 7", new(probe)); err != nil {
		t.Fatalf("legacy enum tolerance changed: %v", err)
	}
	if err := UnmarshalString("", (*probe)(nil)); err == nil {
		t.Fatal("nil accepted")
	}
	if _, err := (&Marshaler{}).MarshalToString((*probe)(nil)); err == nil {
		t.Fatal("nil emitted")
	}
}

func TestStandardUnicodeEscapes(t *testing.T) {
	for _, field := range []string{"checked", "unchecked", "bytes"} {
		for _, escape := range []string{
			`\ud800`, `\udbff`, `\udc00`, `\udfff`, `\U0000D800`, `\U0000DFFF`,
			`\uD83D\uDE00`, `\U0000D83D\U0000DE00`, `\uD83D\U0000DE00`, `\U0000D83D\uDE00`,
		} {
			input := field + `: "` + escape + `"`
			if err := UnmarshalString(input, new(probe)); err == nil {
				t.Errorf("accepted surrogate code point: %s", input)
			}
			if err := proto.UnmarshalText(input, new(probe)); err != nil {
				t.Errorf("legacy surrogate tolerance changed: %s: %v", input, err)
			}
		}
		for _, escape := range []string{`\u0000`, `\uD7FF`, `\uE000`, `\U0001F600`, `\U0010FFFF`, `\uFFFD`} {
			if err := UnmarshalString(field+`: "`+escape+`"`, new(probe)); err != nil {
				t.Errorf("rejected scalar value %s: %v", escape, err)
			}
		}
	}
	var value probe
	if err := UnmarshalString(`bytes: "\xed\xa0\x80"`, &value); err != nil || string(value.Bytes) != "\xed\xa0\x80" {
		t.Fatalf("raw bytes changed: %x %v", value.Bytes, err)
	}
}

func TestStandardUnknownPrinting(t *testing.T) {
	value := &probe{XXX_unrecognized: []byte{56, 7}}
	text, err := (&Marshaler{}).MarshalToString(value)
	if err != nil || text != "" {
		t.Fatalf("unknown fields not dropped: %q %v", text, err)
	}
	text, err = (&Marshaler{PrintUnknownFields: true}).MarshalToString(value)
	if err != nil || !strings.Contains(text, "7: 7") || strings.Contains(text, "/*") {
		t.Fatalf("unknown output: %q %v", text, err)
	}
	if text := proto.MarshalTextString(value); !strings.Contains(text, "/*") {
		t.Fatalf("legacy unknown debug format changed: %q", text)
	}
	value.XXX_unrecognized = []byte{56, 128}
	if _, err := (&Marshaler{PrintUnknownFields: true}).MarshalToString(value); err == nil {
		t.Fatal("malformed unknown wire emitted")
	}
}

func TestStandardUnknownMessagePrinting(t *testing.T) {
	value := &probe{XXX_unrecognized: []byte{58, 2, 8, 111}}
	text, err := (&Marshaler{PrintUnknownFields: true}).MarshalToString(value)
	if err != nil || !strings.Contains(text, "7: {") || !strings.Contains(text, "1: 111") {
		t.Fatalf("unknown message: %q %v", text, err)
	}
	body := []byte{8, 1}
	for i := 0; i < 101; i++ {
		framed := []byte{58}
		framed = append(framed, proto.EncodeVarint(uint64(len(body)))...)
		body = append(framed, body...)
	}
	value.XXX_unrecognized = body
	if _, err := (&Marshaler{PrintUnknownFields: true}).MarshalToString(value); err == nil {
		t.Fatal("unbounded unknown message recursion")
	}
}

type extensionProbe struct{ proto.XXX_InternalExtensions }

func (m *extensionProbe) Reset()       { *m = extensionProbe{} }
func (*extensionProbe) String() string { return "extensionProbe" }
func (*extensionProbe) ProtoMessage()  {}
func (*extensionProbe) ExtensionRangeArray() []proto.ExtensionRange {
	return []proto.ExtensionRange{{Start: 100, End: 110}}
}

var checkedExtension = &proto.ExtensionDesc{ExtendedType: (*extensionProbe)(nil), ExtensionType: (*string)(nil), Field: 100, Name: "text.checked", Tag: "bytes,100,opt,name=checked,proto3,explicit"}
var uncheckedExtension = &proto.ExtensionDesc{ExtendedType: (*extensionProbe)(nil), ExtensionType: (*string)(nil), Field: 101, Name: "text.unchecked", Tag: "bytes,101,opt,name=unchecked,proto3,explicit,utf8_none"}

func init() { proto.RegisterExtension(checkedExtension); proto.RegisterExtension(uncheckedExtension) }
func TestStandardExtensionUTF8Policy(t *testing.T) {
	value := new(extensionProbe)
	if err := proto.SetExtension(value, checkedExtension, proto.String(string([]byte{255}))); err != nil {
		t.Fatal(err)
	}
	if _, err := (&Marshaler{}).MarshalToString(value); err == nil {
		t.Fatal("checked extension emitted invalid UTF-8")
	}
	if err := proto.SetExtension(value, checkedExtension, proto.String("雪")); err != nil {
		t.Fatal(err)
	}
	if err := proto.SetExtension(value, uncheckedExtension, proto.String(string([]byte{255}))); err != nil {
		t.Fatal(err)
	}
	text, err := (&Marshaler{}).MarshalToString(value)
	if err != nil {
		t.Fatal(err)
	}
	var back extensionProbe
	if err := UnmarshalString(text, &back); err != nil {
		t.Fatal(err)
	}
	if !proto.Equal(value, &back) {
		t.Fatal("extension text changed values")
	}
}
