package standard

import (
	"bytes"
	"github.com/gogo/protobuf/proto"
	"github.com/gogo/protobuf/types"
	"strings"
	"testing"
)

type recursiveText struct {
	Child    *recursiveText            `protobuf:"bytes,1,opt,name=child"`
	Children []*recursiveText          `protobuf:"bytes,2,rep,name=children"`
	Map      map[string]*recursiveText `protobuf:"bytes,3,rep,name=map" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value"`
}

func TestStandardTextConfiguredDepth(t *testing.T) {
	for _, depth := range []int{2, 3, 4} {
		input := strings.Repeat("child {", depth-1) + strings.Repeat("}", depth-1)
		var message recursiveText
		err := (&Unmarshaler{RecursionLimit: 3}).UnmarshalString(input, &message)
		if (err == nil) != (depth <= 3) {
			t.Fatalf("decode depth %d: %v", depth, err)
		}
		root := new(recursiveText)
		node := root
		for i := 1; i < depth; i++ {
			node.Child = new(recursiveText)
			node = node.Child
		}
		var out bytes.Buffer
		out.WriteString("prefix")
		err = (&Marshaler{Compact: true, RecursionLimit: 3}).Marshal(&out, root)
		if (err == nil) != (depth <= 3) {
			t.Fatalf("encode depth %d: %v", depth, err)
		}
		if err != nil && out.String() != "prefix" {
			t.Fatal("failed encoder committed partial text")
		}
		if err := proto.UnmarshalText(input, new(recursiveText)); err != nil {
			t.Fatalf("legacy depth behavior changed: %v", err)
		}
	}
	if err := (&Unmarshaler{RecursionLimit: -1}).UnmarshalString("", new(recursiveText)); err == nil {
		t.Fatal("negative decoder limit accepted")
	}
	if _, err := (&Marshaler{RecursionLimit: -1}).MarshalToString(new(recursiveText)); err == nil {
		t.Fatal("negative encoder limit accepted")
	}
}

func TestStandardTextCyclesAndSharedNodes(t *testing.T) {
	for _, link := range []func(*recursiveText){
		func(m *recursiveText) { m.Child = m },
		func(m *recursiveText) { m.Children = []*recursiveText{m} },
		func(m *recursiveText) { m.Map = map[string]*recursiveText{"self": m} },
	} {
		root := new(recursiveText)
		link(root)
		if _, err := (&Marshaler{}).MarshalToString(root); err == nil || !strings.Contains(err.Error(), "cycle") {
			t.Fatalf("cycle not rejected: %v", err)
		}
	}
	leaf := new(recursiveText)
	root := &recursiveText{Child: leaf, Children: []*recursiveText{leaf, leaf}, Map: map[string]*recursiveText{"a": leaf, "b": leaf}}
	text, err := (&Marshaler{RecursionLimit: 3}).MarshalToString(root)
	if err != nil {
		t.Fatalf("shared DAG rejected: %v", err)
	}
	if err := (&Unmarshaler{RecursionLimit: 3}).UnmarshalString(text, new(recursiveText)); err != nil {
		t.Fatal(err)
	}
	if _, err := (&Marshaler{RecursionLimit: 2}).MarshalToString(root); err == nil {
		t.Fatal("map entry omitted from depth")
	}
	if err := (&Unmarshaler{RecursionLimit: 2}).UnmarshalString(text, new(recursiveText)); err == nil {
		t.Fatal("map message depth ignored")
	}
}

func TestStandardReservedAndUnknownDepth(t *testing.T) {
	for _, input := range []string{`removed { a {} }`, `removed [{ a {} }]`} {
		if err := (&Unmarshaler{RecursionLimit: 2}).UnmarshalString(input, new(reservedProbe)); err == nil {
			t.Fatal("reserved depth bypass")
		}
	}
	value := &probe{XXX_unrecognized: []byte{59, 8, 1, 60}}
	if _, err := (&Marshaler{PrintUnknownFields: true, RecursionLimit: 1}).MarshalToString(value); err == nil {
		t.Fatal("unknown group depth bypass")
	}
	if _, err := (&Marshaler{PrintUnknownFields: true, RecursionLimit: 2}).MarshalToString(value); err != nil {
		t.Fatal(err)
	}
}

func (m *recursiveText) Reset()         { *m = recursiveText{} }
func (m *recursiveText) String() string { return proto.CompactTextString(m) }
func (*recursiveText) ProtoMessage()    {}

func TestDefaultTextMessageNestingBound(t *testing.T) {
	input := strings.Repeat("child {", 10000) + strings.Repeat("}", 10000)
	if err := UnmarshalString(input, new(recursiveText)); err == nil {
		t.Fatal("standard text has no bound at the official Go default message depth")
	}
}

type scalarTextMessage int

func (m *scalarTextMessage) Reset()       { *m = 0 }
func (*scalarTextMessage) String() string { return "scalarTextMessage" }
func (*scalarTextMessage) ProtoMessage()  {}
func TestStandardTextNonStructAndNilConfiguration(t *testing.T) {
	if err := UnmarshalString("", new(scalarTextMessage)); err == nil {
		t.Fatal("non-struct decoded")
	}
	if _, err := (&Marshaler{}).MarshalToString(new(scalarTextMessage)); err == nil {
		t.Fatal("non-struct encoded")
	}
	var decoder *Unmarshaler
	if err := decoder.Unmarshal(strings.NewReader(""), new(recursiveText)); err != nil {
		t.Fatal(err)
	}
	var encoder *Marshaler
	if _, err := encoder.MarshalToString(new(recursiveText)); err != nil {
		t.Fatal(err)
	}
}

func TestStandardExpandedAnyDepth(t *testing.T) {
	message := &types.Any{TypeUrl: "type.googleapis.com/google.protobuf.Empty"}
	if _, err := (&Marshaler{ExpandAny: true, RecursionLimit: 1}).MarshalToString(message); err == nil {
		t.Fatal("expanded Any depth ignored")
	}
	text, err := (&Marshaler{ExpandAny: true, RecursionLimit: 2}).MarshalToString(message)
	if err != nil {
		t.Fatal(err)
	}
	if err := (&Unmarshaler{RecursionLimit: 1}).UnmarshalString(text, new(types.Any)); err == nil {
		t.Fatal("expanded Any decode depth ignored")
	}
	if err := (&Unmarshaler{RecursionLimit: 2}).UnmarshalString(text, new(types.Any)); err != nil {
		t.Fatal(err)
	}
}
