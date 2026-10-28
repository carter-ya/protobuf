package descriptor

import (
	"github.com/gogo/protobuf/proto"
	"testing"
)

func TestOptionalDescriptorRoundTrip(t *testing.T) {
	m := &DescriptorProto{
		OneofDecl: []*OneofDescriptorProto{{Name: proto.String("real")}, {Name: proto.String("_value")}},
		Field: []*FieldDescriptorProto{
			{Name: proto.String("real_field"), OneofIndex: proto.Int32(0)},
			{Name: proto.String("value"), OneofIndex: proto.Int32(1), Proto3Optional: proto.Bool(true)},
		},
	}
	b, err := proto.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	var got DescriptorProto
	if err := proto.Unmarshal(b, &got); err != nil {
		t.Fatal(err)
	}
	if !proto.Equal(m, &got) {
		t.Fatal("descriptor data lost")
	}
	if !got.Field[0].IsRealOneof() || got.IsSyntheticOneof(0) {
		t.Fatal("one-field real oneof misclassified")
	}
	if got.Field[1].IsRealOneof() || !got.IsSyntheticOneof(1) {
		t.Fatal("synthetic marker lost")
	}
	if got.IsSyntheticOneof(-1) || got.IsSyntheticOneof(2) {
		t.Fatal("invalid index accepted")
	}
}
