package proto_test

import (
	"github.com/gogo/protobuf/proto"
	"testing"
)

type frozenBytesDefault struct {
	Value []byte `protobuf:"bytes,1,opt,name=value,def=\\001\\377"`
}

func (m *frozenBytesDefault) Reset()       { *m = frozenBytesDefault{} }
func (*frozenBytesDefault) String() string { return "frozenBytesDefault" }
func (*frozenBytesDefault) ProtoMessage()  {}

type privateBytesDefault struct {
	value []byte `protobuf:"bytes,1,opt,name=value,def=\\001\\377"`
}

func (m *privateBytesDefault) Reset()       { *m = privateBytesDefault{} }
func (*privateBytesDefault) String() string { return "privateBytesDefault" }
func (*privateBytesDefault) ProtoMessage()  {}
func (m *privateBytesDefault) XXX_GogoReflectField(index int) interface{} {
	if index == 0 {
		return &m.value
	}
	return nil
}

func TestPrivateDefaultsPreserveFrozenByteConvention(t *testing.T) {
	frozen, private := new(frozenBytesDefault), new(privateBytesDefault)
	proto.SetDefaults(frozen)
	proto.SetDefaults(private)
	if string(frozen.Value) != "\\001\\377" {
		t.Fatal("changed historical exported byte defaults", frozen.Value)
	}
	if string(private.value) != "\x01\xff" {
		t.Fatal("private bytes differ from descriptor default", private.value)
	}
}
