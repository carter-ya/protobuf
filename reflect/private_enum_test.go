package gogoreflect_test

import (
	"bytes"
	"compress/gzip"
	"testing"

	gogo "github.com/gogo/protobuf/proto"
	gogoreflect "github.com/gogo/protobuf/reflect"
	pb "github.com/gogo/protobuf/reflect/internal/testpb"
	v2 "google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
)

type privateClosed struct {
	status           *pb.Status     `protobuf:"varint,1,opt,name=status,enum=live.Status,enum_closed=5:7"`
	child            *privateClosed `protobuf:"bytes,2,opt,name=child"`
	XXX_unrecognized []byte
}

func (m *privateClosed) Reset()                             { *m = privateClosed{} }
func (*privateClosed) String() string                       { return "privateClosed" }
func (*privateClosed) ProtoMessage()                        {}
func (*privateClosed) XXX_GogoReflect()                     {}
func (*privateClosed) Descriptor() ([]byte, []int)          { return privateClosedDescriptor, []int{0} }
func (m *privateClosed) ProtoReflect() protoreflect.Message { return gogoreflect.MessageOf(m) }
func (m *privateClosed) XXX_GogoReflectField(index int) interface{} {
	switch index {
	case 0:
		return &m.status
	case 1:
		return &m.child
	default:
		return nil
	}
}

var privateClosedDescriptor = func() []byte {
	f := &descriptorpb.FileDescriptorProto{
		Name: v2.String("privateclosed.proto"), Package: v2.String("privatehook"), Syntax: v2.String("proto2"),
		Dependency: []string{"reflect/internal/testpb/closed.proto"},
		MessageType: []*descriptorpb.DescriptorProto{{Name: v2.String("Closed"), Field: []*descriptorpb.FieldDescriptorProto{
			{Name: v2.String("status"), Number: v2.Int32(1), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_ENUM.Enum(), TypeName: v2.String(".live.Status")},
			{Name: v2.String("child"), Number: v2.Int32(2), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(), TypeName: v2.String(".privatehook.Closed")},
		}}},
	}
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
	gogo.RegisterFile("privateclosed.proto", b.Bytes())
	gogo.RegisterType((*privateClosed)(nil), "privatehook.Closed")
	return b.Bytes()
}()

func TestPrivateClosedEnumNativeDecode(t *testing.T) {
	for _, data := range [][]byte{{8, 5}, {8, 5, 18, 2, 8, 7}} {
		n := new(privateClosed)
		if err := v2.Unmarshal(data, n); err != nil {
			t.Fatal(err)
		}
		if n.status == nil || *n.status != pb.Status_GOOD {
			t.Fatal("valid closed enum was lost")
		}
		if len(data) > 2 && (n.child == nil || n.child.status == nil || *n.child.status != pb.Status_BAD) {
			t.Fatal("private child lost")
		}
		if !gogo.Equal(n, gogo.Clone(n)) {
			t.Fatal("legacy clone/equal lost private fields")
		}
		copy := new(privateClosed)
		if err := gogo.UnmarshalText(gogo.MarshalTextString(n), copy); err != nil || !gogo.Equal(n, copy) {
			t.Fatal("legacy text private fields", err)
		}
	}
}
