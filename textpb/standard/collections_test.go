package standard

import (
	"github.com/gogo/protobuf/proto"
	"testing"
)

type collectionText struct {
	I       int32            `protobuf:"varint,1,opt,name=i,proto3"`
	Numbers []int32          `protobuf:"varint,2,rep,packed,name=numbers,proto3"`
	Values  map[string]int32 `protobuf:"bytes,3,rep,name=values,proto3" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"varint,2,opt,name=value,proto3"`
}

func (m *collectionText) Reset()         { *m = collectionText{} }
func (m *collectionText) String() string { return proto.CompactTextString(m) }
func (*collectionText) ProtoMessage()    {}
func TestStandardTextCollections(t *testing.T) {
	for _, input := range []string{
		`numbers:[] values:[] i:2`, `numbers:[1,2] values:[{key:'a' value:1},<key:'b' value:2>] i:2`,
		`values [{key:'a' value:1}] i:2`, `values {key:'a' value:1}; i:2`, `values {key:'a'} i:2`,
	} {
		var message collectionText
		if err := UnmarshalString(input, &message); err != nil || message.I != 2 {
			t.Fatalf("%s: %v", input, err)
		}
	}
	for _, input := range []string{`values {key:'a' key:'b' value:1}`, `values {key:'a' value:1 value:2}`, `values:[{} {}]`, `values:[{},]`, `numbers:[1,]`, `i:+1`, `i:+ 1`} {
		if err := UnmarshalString(input, new(collectionText)); err == nil {
			t.Fatalf("accepted %s", input)
		}
	}
	var legacy collectionText
	if err := proto.UnmarshalText(`values {key:'a' key:'b' value:1}`, &legacy); err != nil || legacy.Values["b"] != 1 {
		t.Fatalf("legacy duplicate map behavior changed: %v", err)
	}
	if err := proto.UnmarshalText(`numbers:[]`, new(collectionText)); err == nil {
		t.Fatal("legacy empty-list behavior changed")
	}
	if err := proto.UnmarshalText(`i:+1`, new(collectionText)); err != nil {
		t.Fatal(err)
	}
}
func TestStandardTextWhitespace(t *testing.T) {
	for _, space := range []string{"\v", "\f"} {
		var message collectionText
		if err := UnmarshalString("i:"+space+"1", &message); err != nil || message.I != 1 {
			t.Fatalf("standard whitespace: %v", err)
		}
		if err := proto.UnmarshalText("i:"+space+"1", new(collectionText)); err == nil {
			t.Fatal("legacy whitespace behavior changed")
		}
	}
}

func TestStandardTextIntegerGrammar(t *testing.T) {
	for _, input := range []string{`i:0b1`, `i:0B1`, `i:0o7`, `i:0O7`, `i:1_0`, `i:0x_1`, `i:08`} {
		if err := UnmarshalString(input, new(collectionText)); err == nil {
			t.Fatalf("accepted Go-only integer %s", input)
		}
	}
	for input, want := range map[string]int32{`i:0`: 0, `i:012`: 10, `i:0x10`: 16, `i:-0X10`: -16, `i:- 012`: -10} {
		var got collectionText
		if err := UnmarshalString(input, &got); err != nil || got.I != want {
			t.Fatalf("%s: got %d, want %d: %v", input, got.I, want, err)
		}
	}
}

type boolText struct {
	Flag bool `protobuf:"varint,1,opt,name=flag,proto3"`
}

func (m *boolText) Reset()         { *m = boolText{} }
func (m *boolText) String() string { return proto.CompactTextString(m) }
func (*boolText) ProtoMessage()    {}
func TestStandardTextIntegerBooleans(t *testing.T) {
	for input, want := range map[string]bool{`00`: false, `0x0`: false, `01`: true, `0X1`: true, `0001`: true} {
		var got boolText
		if err := UnmarshalString("flag:"+input, &got); err != nil || got.Flag != want {
			t.Fatalf("%s: got %v want %v: %v", input, got.Flag, want, err)
		}
	}
	for _, input := range []string{`-0`, `+1`, `0b1`, `0o1`, `0_1`, `0x2`, `2`} {
		if err := UnmarshalString("flag:"+input, new(boolText)); err == nil {
			t.Fatalf("accepted %s", input)
		}
	}
	if err := proto.UnmarshalText(`flag:01`, new(boolText)); err == nil {
		t.Fatal("legacy boolean tolerance changed")
	}
}
