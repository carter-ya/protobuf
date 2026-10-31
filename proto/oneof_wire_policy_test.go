package proto

import "testing"

type oneofGroupPolicy struct {
	Choice groupPolicyChoice `protobuf_oneof:"choice"`
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
	return []interface{}{(*groupPolicyMember)(nil)}
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
