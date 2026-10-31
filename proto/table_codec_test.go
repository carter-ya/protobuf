package proto_test

import (
	"bytes"
	"errors"
	"github.com/gogo/protobuf/proto"
	"testing"
)

var compactInfo proto.InternalMessageInfo
var shortcutRecursion = errors.New("recursive codec shortcut")

type compactCodec struct {
	Value        int32 `protobuf:"varint,1,opt,name=value,proto3"`
	MarshalCalls int
	SizeCalls    int
}

func (m *compactCodec) Reset()        { *m = compactCodec{} }
func (*compactCodec) String() string  { return "compact codec" }
func (*compactCodec) ProtoMessage()   {}
func (*compactCodec) XXX_TableCodec() {}
func (m *compactCodec) Marshal() ([]byte, error) {
	m.MarshalCalls++
	if m.MarshalCalls > 1 {
		return nil, shortcutRecursion
	}
	return compactInfo.Marshal(nil, m, false)
}
func (m *compactCodec) Size() int {
	m.SizeCalls++
	if m.SizeCalls > 1 {
		return -1
	}
	return compactInfo.Size(m)
}
func TestTableCodecAvoidsRecursiveShortcuts(t *testing.T) {
	value := &compactCodec{Value: 7}
	encoded, err := value.Marshal()
	if err != nil || !bytes.Equal(encoded, []byte{8, 7}) || value.MarshalCalls != 1 {
		t.Fatal("table wrapper recursively invoked Marshal", encoded, err, value.MarshalCalls)
	}
	if size := value.Size(); size != 2 || value.SizeCalls != 1 {
		t.Fatal("table wrapper recursively invoked Size", size, value.SizeCalls)
	}
}

type historicalShortcut struct {
	Value int32 `protobuf:"varint,1,opt,name=value,proto3"`
}

func (m *historicalShortcut) Reset()                 { *m = historicalShortcut{} }
func (*historicalShortcut) String() string           { return "historical shortcut" }
func (*historicalShortcut) ProtoMessage()            {}
func (*historicalShortcut) Marshal() ([]byte, error) { return []byte{16, 9}, nil }
func (*historicalShortcut) Size() int                { return 2 }
func TestUnmarkedCodecKeepsCustomShortcuts(t *testing.T) {
	var info proto.InternalMessageInfo
	value := &historicalShortcut{Value: 7}
	encoded, err := info.Marshal(nil, value, false)
	if err != nil || !bytes.Equal(encoded, []byte{16, 9}) || info.Size(value) != 2 {
		t.Fatal("unmarked custom codec was bypassed", encoded, err)
	}
}
