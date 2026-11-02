package gogoreflect_test

import (
	"bytes"
	"compress/gzip"
	"reflect"
	"strings"
	"testing"

	gogo "github.com/gogo/protobuf/proto"
	gogoreflect "github.com/gogo/protobuf/reflect"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/encoding/prototext"
	v2 "google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/testing/prototest"
	"google.golang.org/protobuf/types/descriptorpb"
)

// A handwritten fixture isolates the private-storage runtime contract from the
// future OPAQUE generator. Its fields include synthetic and real oneofs.
type privateNode struct {
	count            int32            `protobuf:"varint,1,opt,name=count,proto3"`
	present          *int32           `protobuf:"varint,2,opt,name=present,proto3,oneof,proto3_optional"`
	values           []int32          `protobuf:"varint,3,rep,packed,name=values,proto3"`
	scores           map[string]int32 `protobuf:"bytes,4,rep,name=scores,proto3" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"varint,2,opt,name=value,proto3"`
	child            *privateNode     `protobuf:"bytes,5,opt,name=child,proto3"`
	choice           isPrivateChoice  `protobuf_oneof:"choice"`
	XXX_unrecognized []byte           `json:"-"`
}

type isPrivateChoice interface{ isPrivateChoice() }
type privateText struct {
	Text string `protobuf:"bytes,6,opt,name=text,proto3,oneof"`
}

func (*privateText) isPrivateChoice()                     {}
func (*privateNode) XXX_OneofWrappers() []interface{}     { return []interface{}{(*privateText)(nil)} }
func (m *privateNode) Reset()                             { *m = privateNode{} }
func (*privateNode) String() string                       { return "privateNode" }
func (*privateNode) ProtoMessage()                        {}
func (*privateNode) XXX_GogoReflect()                     {}
func (m *privateNode) ProtoReflect() protoreflect.Message { return gogoreflect.MessageOf(m) }
func (*privateNode) Descriptor() ([]byte, []int)          { return privateDescriptor, []int{0} }
func (m *privateNode) XXX_GogoReflectField(index int) interface{} {
	switch index {
	case 0:
		return &m.count
	case 1:
		return &m.present
	case 2:
		return &m.values
	case 3:
		return &m.scores
	case 4:
		return &m.child
	case 5:
		return &m.choice
	default:
		return nil
	}
}

var privateDescriptor = func() []byte {
	field := func(name string, number int32, kind descriptorpb.FieldDescriptorProto_Type) *descriptorpb.FieldDescriptorProto {
		return &descriptorpb.FieldDescriptorProto{Name: v2.String(name), Number: v2.Int32(number), Type: kind.Enum(), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum()}
	}
	count := field("count", 1, descriptorpb.FieldDescriptorProto_TYPE_INT32)
	present := field("present", 2, descriptorpb.FieldDescriptorProto_TYPE_INT32)
	present.OneofIndex, present.Proto3Optional = v2.Int32(1), v2.Bool(true)
	values := field("values", 3, descriptorpb.FieldDescriptorProto_TYPE_INT32)
	values.Label = descriptorpb.FieldDescriptorProto_LABEL_REPEATED.Enum()
	scores := field("scores", 4, descriptorpb.FieldDescriptorProto_TYPE_MESSAGE)
	scores.Label, scores.TypeName = descriptorpb.FieldDescriptorProto_LABEL_REPEATED.Enum(), v2.String(".privatehook.Node.ScoresEntry")
	child := field("child", 5, descriptorpb.FieldDescriptorProto_TYPE_MESSAGE)
	child.TypeName = v2.String(".privatehook.Node")
	text := field("text", 6, descriptorpb.FieldDescriptorProto_TYPE_STRING)
	text.OneofIndex = v2.Int32(0)
	f := &descriptorpb.FileDescriptorProto{Name: v2.String("privatehook.proto"), Package: v2.String("privatehook"), Syntax: v2.String("proto3"), MessageType: []*descriptorpb.DescriptorProto{{
		Name: v2.String("Node"), Field: []*descriptorpb.FieldDescriptorProto{count, present, values, scores, child, text},
		OneofDecl:  []*descriptorpb.OneofDescriptorProto{{Name: v2.String("choice")}, {Name: v2.String("_present")}},
		NestedType: []*descriptorpb.DescriptorProto{{Name: v2.String("ScoresEntry"), Options: &descriptorpb.MessageOptions{MapEntry: v2.Bool(true)}, Field: []*descriptorpb.FieldDescriptorProto{field("key", 1, descriptorpb.FieldDescriptorProto_TYPE_STRING), field("value", 2, descriptorpb.FieldDescriptorProto_TYPE_INT32)}}},
	}}}
	raw, err := v2.Marshal(f)
	if err != nil {
		panic(err)
	}
	var b bytes.Buffer
	z := gzip.NewWriter(&b)
	if _, err := z.Write(raw); err != nil {
		panic(err)
	}
	if err := z.Close(); err != nil {
		panic(err)
	}
	gogo.RegisterFile("privatehook.proto", b.Bytes())
	gogo.RegisterType((*privateNode)(nil), "privatehook.Node")
	return b.Bytes()
}()

func TestPrivateStorageLiveContract(t *testing.T) {
	m := &privateNode{}
	prototest.Message{}.Test(t, m.ProtoReflect().Type())
	r := m.ProtoReflect()
	fields := r.Descriptor().Fields()
	r.Set(fields.ByNumber(1), protoreflect.ValueOfInt32(42))
	if m.count != 42 {
		t.Fatal("reflection wrote a detached scalar")
	}
	m.present = gogo.Int32(0)
	if !r.Has(fields.ByNumber(2)) {
		t.Fatal("lost zero presence")
	}
	r.Mutable(fields.ByNumber(3)).List().Append(protoreflect.ValueOfInt32(7))
	r.Mutable(fields.ByNumber(4)).Map().Set(protoreflect.ValueOfString("key").MapKey(), protoreflect.ValueOfInt32(8))
	r.Mutable(fields.ByNumber(5)).Message().Set(fields.ByNumber(1), protoreflect.ValueOfInt32(9))
	r.Set(fields.ByNumber(6), protoreflect.ValueOfString("selected"))
	if m.values[0] != 7 || m.scores["key"] != 8 || m.child.count != 9 || m.choice.(*privateText).Text != "selected" {
		t.Fatal("composite/oneof storage is detached")
	}
	if r.Interface() != m || reflect.TypeOf(v2.Clone(m)) != reflect.TypeOf(m) {
		t.Fatal("native identity lost")
	}
	for _, codec := range []struct {
		name      string
		marshal   func(v2.Message) ([]byte, error)
		unmarshal func([]byte, v2.Message) error
	}{
		{"binary", v2.Marshal, v2.Unmarshal}, {"json", protojson.Marshal, protojson.Unmarshal}, {"text", prototext.Marshal, prototext.Unmarshal},
	} {
		t.Run(codec.name, func(t *testing.T) {
			b, err := codec.marshal(m)
			if err != nil {
				t.Fatal(err)
			}
			n := &privateNode{}
			if err = codec.unmarshal(b, n); err != nil {
				t.Fatal(err)
			}
			if !v2.Equal(m, n) {
				t.Fatal("round trip differs")
			}
		})
	}
	r.Clear(fields.ByNumber(6))
	if m.choice != nil {
		t.Fatal("oneof clear did not reach native storage")
	}
	var zero *privateNode
	if zero.ProtoReflect().IsValid() || zero.ProtoReflect().Has(fields.ByNumber(1)) {
		t.Fatal("typed nil is valid")
	}
}

type detachedNode privateNode

func (m *detachedNode) Reset()                         { *m = detachedNode{} }
func (*detachedNode) String() string                   { return "detachedNode" }
func (*detachedNode) ProtoMessage()                    {}
func (*detachedNode) Descriptor() ([]byte, []int)      { return privateDescriptor, []int{0} }
func (*detachedNode) XXX_OneofWrappers() []interface{} { return []interface{}{(*privateText)(nil)} }
func (m *detachedNode) XXX_GogoReflectField(index int) interface{} {
	if index == 0 {
		return new(int32)
	}
	return (*privateNode)(m).XXX_GogoReflectField(index)
}

func TestPrivateStorageRejectsDetachedAccessor(t *testing.T) {
	if _, err := gogoreflect.Wrap(&detachedNode{}); err == nil || !strings.Contains(err.Error(), "detached") {
		t.Fatalf("expected detached-storage diagnostic, got %v", err)
	}
}
