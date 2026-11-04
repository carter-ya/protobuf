package standard

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/gogo/protobuf/proto"
)

type requiredText struct {
	Name *string `protobuf:"bytes,1,req,name=name"`
	Data []byte  `protobuf:"bytes,2,req,name=data"`
}

func (m *requiredText) Reset()         { *m = requiredText{} }
func (m *requiredText) String() string { return proto.CompactTextString(m) }
func (*requiredText) ProtoMessage()    {}

type nonnullableRequiredText struct {
	Value int32  `protobuf:"varint,1,req,name=value,proto3"`
	Data  []byte `protobuf:"bytes,2,req,name=data,proto3"`
}

func (m *nonnullableRequiredText) Reset()         { *m = nonnullableRequiredText{} }
func (m *nonnullableRequiredText) String() string { return proto.CompactTextString(m) }
func (*nonnullableRequiredText) ProtoMessage()    {}

type presenceChoice interface{ presenceChoice() }
type presenceMember struct {
	Message *probe `protobuf:"bytes,4,opt,name=message,proto3,oneof"`
}

func (*presenceMember) presenceChoice() {}

type presenceText struct {
	Children         []*probe                 `protobuf:"bytes,1,rep,name=children,proto3"`
	Values           map[string]*probe        `protobuf:"bytes,2,rep,name=values,proto3" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
	Required         *requiredText            `protobuf:"bytes,3,opt,name=required"`
	Choice           presenceChoice           `protobuf_oneof:"choice"`
	RequiredChildren []*requiredText          `protobuf:"bytes,5,rep,name=required_children"`
	RequiredValues   map[string]*requiredText `protobuf:"bytes,6,rep,name=required_values" protobuf_key:"bytes,1,opt,name=key" protobuf_val:"bytes,2,opt,name=value"`
}

func (m *presenceText) Reset()                         { *m = presenceText{} }
func (m *presenceText) String() string                 { return proto.CompactTextString(m) }
func (*presenceText) ProtoMessage()                    {}
func (*presenceText) XXX_OneofWrappers() []interface{} { return []interface{}{(*presenceMember)(nil)} }

func TestStandardTextNilMessagesAreEmpty(t *testing.T) {
	for _, compact := range []bool{false, true} {
		for _, message := range []*presenceText{
			{Children: []*probe{nil, new(probe)}}, {Values: map[string]*probe{"a": nil, "b": new(probe)}},
			{Choice: &presenceMember{}}, {Choice: (*presenceMember)(nil)},
		} {
			output, err := (&Marshaler{Compact: compact}).MarshalToString(message)
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(output, "nil") {
				t.Fatalf("invalid nil marker %q", output)
			}
			var back presenceText
			if err := UnmarshalString(output, &back); err != nil {
				t.Fatalf("invalid standard output %q: %v", output, err)
			}
			if len(back.Children) != len(message.Children) || len(back.Values) != len(message.Values) {
				t.Fatal("empty message presence lost")
			}
			if message.Choice != nil {
				member := message.Choice.(*presenceMember)
				if member != nil {
					if back.Choice == nil || back.Choice.(*presenceMember).Message == nil {
						t.Fatal("selected nil message lost oneof presence")
					}
				} else if back.Choice != nil {
					t.Fatal("nil wrapper selected oneof")
				}
			}
		}
	}
	legacy := proto.MarshalTextString(&presenceText{Children: []*probe{nil}, Choice: &presenceMember{}})
	if !strings.Contains(legacy, "<nil>") || !strings.Contains(legacy, "/* nil */") {
		t.Fatalf("legacy nil markers changed: %q", legacy)
	}
}
func TestStandardTextRequiredPresence(t *testing.T) {
	for _, message := range []proto.Message{
		new(requiredText), &requiredText{Name: proto.String("")},
		&presenceText{Required: new(requiredText)}, &presenceText{RequiredChildren: []*requiredText{nil}},
		&presenceText{RequiredValues: map[string]*requiredText{"a": nil}},
	} {
		var out bytes.Buffer
		out.WriteString("untouched")
		err := (&Marshaler{}).Marshal(&out, message)
		if _, ok := err.(*proto.RequiredNotSetError); !ok {
			t.Fatalf("missing required field must error: %T %v", message, err)
		}
		if out.String() != "untouched" {
			t.Fatal("partial output escaped buffer")
		}
	}
	for _, message := range []proto.Message{&requiredText{Name: proto.String(""), Data: []byte{}}, &nonnullableRequiredText{Data: []byte{}}} {
		output, err := (&Marshaler{}).MarshalToString(message)
		if err != nil {
			t.Fatal(err)
		}
		back := proto.Clone(message)
		back.Reset()
		if err := UnmarshalString(output, back); err != nil {
			t.Fatalf("required zero presence lost in %q: %v", output, err)
		}
	}
	if _, err := (&Marshaler{}).MarshalToString(new(presenceText)); err != nil {
		t.Fatal("absent optional required-message field must remain absent", err)
	}
	if text := proto.MarshalTextString(new(requiredText)); text != "" {
		t.Fatalf("legacy partial text behavior changed: %q", text)
	}
}

func TestStandardNilMessagesObserveRecursionLimit(t *testing.T) {
	for _, test := range []struct {
		message *presenceText
		depth   int
	}{
		{&presenceText{Children: []*probe{nil}}, 2},
		{&presenceText{Values: map[string]*probe{"a": nil}}, 3},
		{&presenceText{Choice: &presenceMember{}}, 2},
	} {
		if _, err := (&Marshaler{RecursionLimit: test.depth - 1}).MarshalToString(test.message); err == nil {
			t.Fatal("nil normalization bypassed recursion budget")
		}
		if _, err := (&Marshaler{RecursionLimit: test.depth}).MarshalToString(test.message); err != nil {
			t.Fatal(err)
		}
	}
}

type nilMappedText struct {
	Times []*time.Time `protobuf:"bytes,1,rep,name=times,stdtime"`
}

func (m *nilMappedText) Reset()         { *m = nilMappedText{} }
func (m *nilMappedText) String() string { return proto.CompactTextString(m) }
func (*nilMappedText) ProtoMessage()    {}
func TestStandardNilMappedValueReturnsError(t *testing.T) {
	var out bytes.Buffer
	if err := (&Marshaler{}).Marshal(&out, &nilMappedText{Times: []*time.Time{nil}}); err == nil {
		t.Fatal("nil mapped value must require an explicit representation")
	}
	if out.Len() != 0 {
		t.Fatal("invalid mapped value wrote partial output")
	}
}
