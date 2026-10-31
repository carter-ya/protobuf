package standard

import (
	"bytes"
	"compress/gzip"
	"strings"
	"testing"

	"github.com/gogo/protobuf/proto"
	descriptor "github.com/gogo/protobuf/protoc-gen-gogo/descriptor"
)

type reservedProbe struct {
	Known    int32             `protobuf:"varint,1,opt,name=known,proto3"`
	Child    *reservedNested   `protobuf:"bytes,2,opt,name=child"`
	Children []*reservedNested `protobuf:"bytes,3,rep,name=children"`
}

type reservedNested struct {
	Known int32 `protobuf:"varint,1,opt,name=known,proto3"`
}

func (m *reservedProbe) Reset()                     { *m = reservedProbe{} }
func (m *reservedProbe) String() string             { return proto.CompactTextString(m) }
func (*reservedProbe) ProtoMessage()                {}
func (*reservedProbe) Descriptor() ([]byte, []int)  { return reservedDescriptor, []int{0} }
func (m *reservedNested) Reset()                    { *m = reservedNested{} }
func (m *reservedNested) String() string            { return proto.CompactTextString(m) }
func (*reservedNested) ProtoMessage()               {}
func (*reservedNested) Descriptor() ([]byte, []int) { return reservedDescriptor, []int{0, 0} }

var reservedDescriptor = func() []byte {
	file := &descriptor.FileDescriptorProto{MessageType: []*descriptor.DescriptorProto{{
		Name: proto.String("ReservedProbe"), ReservedName: []string{"removed"},
		NestedType: []*descriptor.DescriptorProto{{Name: proto.String("Nested"), ReservedName: []string{"nested_removed"}}},
	}}}
	wire, err := proto.Marshal(file)
	if err != nil {
		panic(err)
	}
	var out bytes.Buffer
	writer := gzip.NewWriter(&out)
	if _, err := writer.Write(wire); err != nil {
		panic(err)
	}
	if err := writer.Close(); err != nil {
		panic(err)
	}
	return out.Bytes()
}()

func TestStandardReservedNames(t *testing.T) {
	for _, value := range []string{
		`true`, `- # sign\n123`, `0.123`, `ENUM_VALUE`, `"hello" 'world'`,
		`{ a: 123 }`, `< a: 123 >`, `[-123, 456]`, `[0.123, 1e-10]`, `["hello", "world"]`,
		`[{ a { b: 1 } }, < b: [1, 2] >]`, `[]`,
	} {
		value = strings.ReplaceAll(value, `\n`, "\n")
		var got reservedProbe
		if err := UnmarshalString("removed: "+value+"; known: 7", &got); err != nil || got.Known != 7 {
			t.Errorf("skip %s: %v (%v)", value, err, got)
		}
	}
	for _, input := range []string{`removed { a: 1 } known: 7`, `child { nested_removed: [1, 2] known: 7 }`} {
		if err := UnmarshalString(input, new(reservedProbe)); err != nil {
			t.Error(err)
		}
	}
	for _, input := range []string{
		`unknown: 1`, `nested_removed: 1`, `child { removed: 1 }`,
		`removed 1`, `removed:`, `removed: { a: 1 >`, `removed: [1,]`,
		`removed: [1 2]`, `removed: { a 1 }`, `removed: +1`, `removed: -NAME`,
		`removed: "\uD800"`,
	} {
		if err := UnmarshalString(input, new(reservedProbe)); err == nil {
			t.Errorf("accepted %s", input)
		}
	}
	if err := proto.UnmarshalText(`removed: 1`, new(reservedProbe)); err == nil {
		t.Fatal("legacy unknown-field rejection changed")
	}
	deep := "removed: " + strings.Repeat("{ a: ", 101) + "1" + strings.Repeat("}", 101)
	if err := UnmarshalString(deep, new(reservedProbe)); err == nil {
		t.Fatal("unbounded reserved nesting")
	}
}

type brokenDescriptorProbe struct{ probe }

func (*brokenDescriptorProbe) Descriptor() ([]byte, []int) { return []byte("not gzip"), []int{0} }

type badPathProbe struct{ probe }

func (*badPathProbe) Descriptor() ([]byte, []int) { return reservedDescriptor, []int{0, 1} }

type emptyPathProbe struct{ probe }

func (*emptyPathProbe) Descriptor() ([]byte, []int) { return reservedDescriptor, nil }

func TestStandardInvalidTextDescriptor(t *testing.T) {
	for _, message := range []proto.Message{new(brokenDescriptorProbe), new(badPathProbe), new(emptyPathProbe)} {
		if err := UnmarshalString("", message); err == nil {
			t.Errorf("accepted invalid descriptor for %T", message)
		}
	}
}

func TestStandardReservedNamesConcurrent(t *testing.T) {
	for i := 0; i < 16; i++ {
		t.Run("parse", func(t *testing.T) {
			t.Parallel()
			for j := 0; j < 16; j++ {
				var message reservedProbe
				if err := UnmarshalString(`removed { a: 1 } child { nested_removed: [] known: 3 }`, &message); err != nil || message.Child == nil || message.Child.Known != 3 {
					t.Fatalf("concurrent schema: %v", err)
				}
			}
		})
	}
}

func TestStandardReservedMessageListWithoutColon(t *testing.T) {
	if err := UnmarshalString(`children [{ known: 1 }]`, new(reservedProbe)); err != nil {
		t.Fatalf("declared message-list spelling: %v", err)
	}
	for _, input := range []string{
		`removed [{ a: 1 }] known: 7`, `removed [< a: 1 >, {}] known: 7`,
		`removed [] known: 7`, `removed { children [{ a: 1 }] } known: 7`,
	} {
		var message reservedProbe
		if err := UnmarshalString(input, &message); err != nil || message.Known != 7 {
			t.Errorf("reserved message-list %q: %v", input, err)
		}
	}
	for _, input := range []string{`removed [1]`, `removed ["s"]`, `removed [{}, 1]`, `removed [{},]`, `removed [[{}]]`} {
		if err := UnmarshalString(input, new(reservedProbe)); err == nil {
			t.Errorf("accepted colonless non-message list %q", input)
		}
	}
}
