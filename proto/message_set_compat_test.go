package proto_test

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/gogo/protobuf/proto"
	pb "github.com/gogo/protobuf/proto/test_proto"
)

type messageSetPayload struct {
	A *int32 `protobuf:"varint,1,opt,name=a"`
	B *int32 `protobuf:"varint,2,opt,name=b"`
}

func (m *messageSetPayload) Reset()         { *m = messageSetPayload{} }
func (m *messageSetPayload) String() string { return proto.CompactTextString(m) }
func (*messageSetPayload) ProtoMessage()    {}

var messageSetExtension = &proto.ExtensionDesc{ExtendedType: (*pb.MyMessageSet)(nil), ExtensionType: (*messageSetPayload)(nil), Field: 9876543, Name: "test.messageset.compat", Tag: "bytes,9876543,opt,name=compat"}

func init() { proto.RegisterExtension(messageSetExtension) }
func messageSetOrdinary(id int32, body []byte) []byte {
	wire := proto.EncodeVarint(uint64(id)<<3 | 2)
	wire = append(wire, proto.EncodeVarint(uint64(len(body)))...)
	return append(wire, body...)
}
func messageSetGroup(id int32, body []byte) []byte {
	wire := append([]byte{11, 16}, proto.EncodeVarint(uint64(id))...)
	wire = append(wire, 26)
	wire = append(wire, proto.EncodeVarint(uint64(len(body)))...)
	wire = append(wire, body...)
	return append(wire, 12)
}
func TestMessageSetOrdinaryInput(t *testing.T) {
	first := messageSetOrdinary(messageSetExtension.Field, []byte{8, 1, 16, 3})
	second := messageSetGroup(messageSetExtension.Field, []byte{8, 2})
	for _, tc := range []struct {
		input []byte
		want  int32
	}{
		{first, 1}, {append(append([]byte{}, first...), second...), 2}, {append(append([]byte{}, second...), first...), 1},
	} {
		var message pb.MyMessageSet
		if err := proto.Unmarshal(tc.input, &message); err != nil {
			t.Fatal(err)
		}
		if !proto.HasExtension(&message, messageSetExtension) {
			t.Fatal("ordinary MessageSet extension lost")
		}
		value, err := proto.GetExtension(&message, messageSetExtension)
		if err != nil {
			t.Fatal(err)
		}
		payload := value.(*messageSetPayload)
		if payload.A == nil || *payload.A != tc.want || payload.B == nil || *payload.B != 3 {
			t.Fatalf("merge order lost: %v", payload)
		}
		wire, err := proto.Marshal(&message)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.HasPrefix(wire, []byte{11, 16}) {
			t.Fatalf("noncanonical MessageSet output %x", wire)
		}
	}
}
func TestMessageSetMergeDecodedExtension(t *testing.T) {
	for _, decodedByGet := range []bool{false, true} {
		var message pb.MyMessageSet
		if err := proto.SetExtension(&message, messageSetExtension, &messageSetPayload{A: proto.Int32(1), B: proto.Int32(3)}); err != nil {
			t.Fatal(err)
		}
		if decodedByGet {
			wire, err := proto.Marshal(&message)
			if err != nil {
				t.Fatal(err)
			}
			if err := proto.Unmarshal(wire, &message); err != nil {
				t.Fatal(err)
			}
			if _, err := proto.GetExtension(&message, messageSetExtension); err != nil {
				t.Fatal(err)
			}
		}
		if err := proto.UnmarshalMerge(messageSetGroup(messageSetExtension.Field, []byte{8, 2}), &message); err != nil {
			t.Fatal(err)
		}
		value, err := proto.GetExtension(&message, messageSetExtension)
		if err != nil {
			t.Fatal(err)
		}
		payload := value.(*messageSetPayload)
		if payload.A == nil || *payload.A != 2 || payload.B == nil || *payload.B != 3 {
			t.Fatalf("decoded merge lost: %v", payload)
		}
	}
}
func TestMessageSetOrdinaryUnknownAndMalformed(t *testing.T) {
	body := []byte{0, 1} // intentionally not a valid known message
	var message pb.MyMessageSet
	if err := proto.Unmarshal(messageSetOrdinary(12345, body), &message); err != nil {
		t.Fatal(err)
	}
	wire, err := proto.Marshal(&message)
	if err != nil || !bytes.Equal(wire, messageSetGroup(12345, body)) {
		t.Fatalf("unknown payload changed: %x %v", wire, err)
	}
	for _, wire := range [][]byte{{18, 3, 1}, {18, 128}, {11, 16, 1, 26, 0, 20}} {
		if err := proto.Unmarshal(wire, new(pb.MyMessageSet)); err == nil {
			t.Fatalf("malformed input accepted: %x", wire)
		}
	}
}

func TestMessageSetLargeTypeIDMerging(t *testing.T) {
	for _, id := range []int32{536870911, 536870912, 2147483646} {
		desc := &proto.ExtensionDesc{ExtendedType: (*pb.MyMessageSet)(nil), ExtensionType: (*messageSetPayload)(nil), Field: id, Name: fmt.Sprintf("test.messageset.large%d", id), Tag: fmt.Sprintf("bytes,%d,opt,name=large", id)}
		first, second := messageSetGroup(id, []byte{8, 1}), messageSetGroup(id, []byte{16, 2})
		for _, incremental := range []bool{false, true} {
			var message pb.MyMessageSet
			if incremental {
				if err := proto.Unmarshal(first, &message); err != nil {
					t.Fatal(err)
				}
				if err := proto.UnmarshalMerge(second, &message); err != nil {
					t.Fatalf("type ID %d: %v", id, err)
				}
			} else {
				if err := proto.Unmarshal(append(append([]byte{}, first...), second...), &message); err != nil {
					t.Fatalf("type ID %d: %v", id, err)
				}
			}
			value, err := proto.GetExtension(&message, desc)
			if err != nil {
				t.Fatal(err)
			}
			payload := value.(*messageSetPayload)
			if payload.A == nil || *payload.A != 1 || payload.B == nil || *payload.B != 2 {
				t.Fatalf("large type ID merge lost: %v", payload)
			}
			wire, err := proto.Marshal(&message)
			if err != nil || !bytes.Equal(wire, messageSetGroup(id, []byte{8, 1, 16, 2})) {
				t.Fatalf("large type ID output %x: %v", wire, err)
			}
		}
	}
}
