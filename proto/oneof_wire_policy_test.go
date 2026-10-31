package proto

import (
	"bytes"
	"testing"
)

type oneofGroupPolicy struct {
	Choice           groupPolicyChoice `protobuf_oneof:"choice"`
	XXX_unrecognized []byte
}
type groupPolicyChoice interface{ groupPolicyChoice() }
type groupPolicyMember struct {
	Group *groupPolicyPayload `protobuf:"group,1,opt,name=group,proto3,oneof,explicit"`
}
type groupPolicyPayload struct {
	A *int32 `protobuf:"varint,1,opt,name=a"`
	B *int32 `protobuf:"varint,2,opt,name=b"`
}

func (*groupPolicyMember) groupPolicyChoice() {}
func (m *oneofGroupPolicy) Reset()            { *m = oneofGroupPolicy{} }
func (m *oneofGroupPolicy) String() string    { return CompactTextString(m) }
func (*oneofGroupPolicy) ProtoMessage()       {}
func (*oneofGroupPolicy) XXX_OneofWrappers() []interface{} {
	return []interface{}{(*groupPolicyMember)(nil), (*scalarPolicyMember)(nil)}
}
func (m *groupPolicyPayload) Reset()         { *m = groupPolicyPayload{} }
func (m *groupPolicyPayload) String() string { return CompactTextString(m) }
func (*groupPolicyPayload) ProtoMessage()    {}

func TestOneofGroupWirePolicy(t *testing.T) {
	for _, wire := range [][]byte{
		{11, 8, 1, 12, 11, 16, 2, 12},
		{11, 8, 1, 12, 10, 0, 11, 16, 2, 12}, // wrong length-delimited field is skipped
	} {
		var message oneofGroupPolicy
		if err := Unmarshal(wire, &message); err != nil {
			t.Fatal(err)
		}
		member, ok := message.Choice.(*groupPolicyMember)
		if !ok || member.Group == nil || member.Group.A == nil || *member.Group.A != 1 || member.Group.B == nil || *member.Group.B != 2 {
			t.Fatalf("group wire policy lost fields: %v", &message)
		}
	}
}

type scalarPolicyMember struct {
	Value int32 `protobuf:"varint,2,opt,name=scalar,proto3,oneof"`
}

func (*scalarPolicyMember) groupPolicyChoice() {}
func TestOneofScalarWrongWirePreservesInputAndChoice(t *testing.T) {
	for _, unknown := range [][]byte{{17, 1, 2, 3, 4, 5, 6, 7, 8}, {18, 0}, {19, 24, 1, 20}, {21, 1, 2, 3, 4}} {
		for _, selected := range []bool{false, true} {
			input := append([]byte{}, unknown...)
			if selected {
				input = append([]byte{16, 5}, input...)
			}
			var got oneofGroupPolicy
			if err := Unmarshal(input, &got); err != nil {
				t.Fatalf("%x: %v", input, err)
			}
			if selected {
				value, ok := got.Choice.(*scalarPolicyMember)
				if !ok || value.Value != 5 {
					t.Fatalf("wrong wire changed oneof: %v", &got)
				}
			} else if got.Choice != nil {
				t.Fatal("wrong wire selected oneof")
			}
			if !bytes.Equal(got.XXX_unrecognized, unknown) {
				t.Fatalf("lost unknown %x", got.XXX_unrecognized)
			}
		}
	}
}
