package jsonpb

import (
	"bytes"
	"fmt"
	"math"
	"testing"

	pb "github.com/gogo/protobuf/jsonpb/jsonpb_test_proto"
	"github.com/gogo/protobuf/proto"
	"github.com/gogo/protobuf/types"
)

func TestIntegerJSONForms(t *testing.T) {
	for _, input := range []string{"1e2", `"1e2"`, "100.000", `"\u0031e2"`, "1000e-1", "0.01e4"} {
		value := new(types.Int32Value)
		if err := UnmarshalString(input, value); err != nil || value.Value != 100 {
			t.Fatalf("%s: %v %v", input, value, err)
		}
	}
	for _, input := range []string{"1.1", "1e-1", "1e2147483647", "1e-2147483648", `"01"`, "2147483648", "-2147483649"} {
		if err := UnmarshalString(input, new(types.Int32Value)); err == nil {
			t.Fatalf("invalid int32 accepted: %s", input)
		}
	}
	for _, tc := range []struct {
		input string
		want  int64
	}{{"9223372036854775807.0", math.MaxInt64}, {"-9223372036854775808e0", math.MinInt64}, {"0e999999999999999999999", 0}, {"-0.000e-999999999999999", 0}} {
		value := new(types.Int64Value)
		if err := UnmarshalString(tc.input, value); err != nil || value.Value != tc.want {
			t.Fatalf("%s: %v %v", tc.input, value, err)
		}
	}
	var unsigned types.UInt64Value
	if err := UnmarshalString(`"18446744073709551615.0"`, &unsigned); err != nil || unsigned.Value != math.MaxUint64 {
		t.Fatalf("uint64 max: %v %v", unsigned, err)
	}
	for _, input := range []string{"18446744073709551616", "-1", "1.5"} {
		if err := UnmarshalString(input, new(types.UInt64Value)); err == nil {
			t.Fatal(input)
		}
	}
}

func TestBytesJSONAlphabets(t *testing.T) {
	for _, encoded := range []string{"/w==", "/w", "_w==", "_w", ""} {
		value := new(types.BytesValue)
		if err := UnmarshalString(fmt.Sprintf("%q", encoded), value); err != nil {
			t.Fatalf("%q: %v", encoded, err)
		}
		expected := []byte{255}
		if encoded == "" {
			expected = nil
		}
		if !bytes.Equal(value.Value, expected) {
			t.Fatalf("%q: %x", encoded, value.Value)
		}
	}
	for _, encoded := range []string{"!", "a", "_w=", "/w===="} {
		if err := UnmarshalString(fmt.Sprintf("%q", encoded), new(types.BytesValue)); err == nil {
			t.Fatal(encoded)
		}
	}
	var value pb.Simple
	if err := UnmarshalString(`{"oInt32":"1e2","oBytes":"_w"}`, &value); err != nil || value.GetOInt32() != 100 || !bytes.Equal(value.OBytes, []byte{255}) {
		t.Fatalf("nested: %v %v", value, err)
	}
}

type escapedEnumProbe struct {
	Syntax types.Syntax `protobuf:"varint,1,opt,name=syntax,proto3,enum=google.protobuf.Syntax" json:"syntax,omitempty"`
}

func (m *escapedEnumProbe) Reset()         { *m = escapedEnumProbe{} }
func (m *escapedEnumProbe) String() string { return proto.CompactTextString(m) }
func (*escapedEnumProbe) ProtoMessage()    {}
func TestEscapedLegacyEnumName(t *testing.T) {
	var value escapedEnumProbe
	if err := UnmarshalString(`{"syntax":"\u0053YNTAX_PROTO3"}`, &value); err != nil || value.Syntax != types.Syntax_SYNTAX_PROTO3 {
		t.Fatalf("%v %v", value, err)
	}
	if err := UnmarshalString(`{"syntax":"UNKNOWN"}`, &value); err == nil {
		t.Fatal("unknown enum accepted")
	}
}
