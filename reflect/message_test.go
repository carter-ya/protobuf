package gogoreflect_test

import (
	"bytes"
	"math"
	"sync"
	"testing"

	gogo "github.com/gogo/protobuf/proto"
	live "github.com/gogo/protobuf/reflect"
	pb "github.com/gogo/protobuf/reflect/internal/testpb"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/encoding/prototext"
	v2 "google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/testing/prototest"
	"google.golang.org/protobuf/types/dynamicpb"
)

func view(t *testing.T, m *pb.Node) protoreflect.Message {
	t.Helper()
	p, e := live.Wrap(m)
	if e != nil {
		t.Fatal(e)
	}
	return p.ProtoReflect()
}
func field(m protoreflect.Message, n string) protoreflect.FieldDescriptor {
	return m.Descriptor().Fields().ByName(protoreflect.Name(n))
}
func TestLiveFieldsAndCollections(t *testing.T) {
	native := new(pb.Node)
	m := view(t, native)
	count := field(m, "count")
	m.Set(count, protoreflect.ValueOfInt32(7))
	if native.Count != 7 {
		t.Fatal(native)
	}
	native.Count = 9
	if m.Get(count).Int() != 9 {
		t.Fatal("not live")
	}
	present := field(m, "present")
	m.Set(present, protoreflect.ValueOfInt32(0))
	if native.Present == nil || !m.Has(present) {
		t.Fatal("presence")
	}
	m.Clear(present)
	if native.Present != nil {
		t.Fatal("clear")
	}
	child := field(m, "child")
	if m.Get(child).Message().IsValid() {
		t.Fatal("absent child")
	}
	c := m.Mutable(child).Message()
	c.Set(count, protoreflect.ValueOfInt32(3))
	if native.Child.Count != 3 {
		t.Fatal("child detached")
	}
	native.Child.Count = 4
	if c.Get(count).Int() != 4 {
		t.Fatal("child not live")
	}
	values := field(m, "values")
	list := m.Mutable(values).List()
	list.Append(protoreflect.ValueOfInt32(2))
	native.Values = append(native.Values, 5)
	if list.Len() != 2 {
		t.Fatal("slice header stale")
	}
	list.Set(1, protoreflect.ValueOfInt32(6))
	list.Truncate(1)
	if len(native.Values) != 1 || native.Values[0] != 2 {
		t.Fatal(native.Values)
	}
	children := m.Mutable(field(m, "children")).List()
	children.AppendMutable().Message().Set(count, protoreflect.ValueOfInt32(11))
	if native.Children[0].Count != 11 {
		t.Fatal("list child detached")
	}
	scores := m.Mutable(field(m, "scores")).Map()
	key := protoreflect.ValueOfString("x").MapKey()
	scores.Set(key, protoreflect.ValueOfInt32(13))
	if native.Scores["x"] != 13 {
		t.Fatal("map detached")
	}
	native.Scores["y"] = 17
	if scores.Len() != 2 {
		t.Fatal("map not live")
	}
	scores.Clear(key)
	if _, ok := native.Scores["x"]; ok {
		t.Fatal("map clear")
	}
	nodes := m.Mutable(field(m, "nodes")).Map()
	nodes.Mutable(key).Message().Set(count, protoreflect.ValueOfInt32(19))
	if native.Nodes["x"].Count != 19 {
		t.Fatal("map child detached")
	}
	number := field(m, "number")
	selected := field(m, "selected")
	m.Set(number, protoreflect.ValueOfInt32(0))
	if !m.Has(number) || m.WhichOneof(number.ContainingOneof()) != number {
		t.Fatal("oneof zero")
	}
	m.Mutable(selected).Message().Set(count, protoreflect.ValueOfInt32(23))
	if native.GetSelected().Count != 23 || m.Has(number) {
		t.Fatal("oneof replacement")
	}
	m.Clear(number)
	if !m.Has(selected) {
		t.Fatal("cleared other member")
	}
}
func TestOfficialOperations(t *testing.T) {
	original := &pb.Node{Count: 4, Name: "hello", Present: gogo.Int32(0), Data: []byte{0, 1}, Child: &pb.Node{Count: 2}, Children: []*pb.Node{{Count: 3}}, Values: []int32{1, 2}, Nodes: map[string]*pb.Node{"a": {Name: "nested"}}, Scores: map[string]int32{"a": 5}, Choice: &pb.Node_Number{Number: 0}, Large: ^uint64(0)}
	wrapped := view(t, original).Interface()
	wire, err := (v2.MarshalOptions{Deterministic: true}).Marshal(wrapped)
	if err != nil {
		t.Fatal(err)
	}
	if len(wire) != v2.Size(wrapped) {
		t.Fatal("size")
	}
	fromGogo := new(pb.Node)
	if err = gogo.Unmarshal(wire, fromGogo); err != nil || !gogo.Equal(original, fromGogo) {
		t.Fatal(err, fromGogo)
	}
	clone := v2.Clone(wrapped)
	if !v2.Equal(wrapped, clone) {
		t.Fatal("clone")
	}
	clone.ProtoReflect().Set(field(clone.ProtoReflect(), "name"), protoreflect.ValueOfString("changed"))
	if original.Name != "hello" {
		t.Fatal("clone aliases")
	}
	for _, format := range []string{"binary", "json", "text"} {
		t.Run(format, func(t *testing.T) {
			dest := new(pb.Node)
			p := view(t, dest).Interface()
			switch format {
			case "binary":
				err = v2.Unmarshal(wire, p)
			case "json":
				var b []byte
				b, err = protojson.Marshal(wrapped)
				if err == nil {
					err = protojson.Unmarshal(b, p)
				}
			case "text":
				var b []byte
				b, err = prototext.Marshal(wrapped)
				if err == nil {
					err = prototext.Unmarshal(b, p)
				}
			}
			if err != nil || !gogo.Equal(original, dest) {
				t.Fatal(err, dest)
			}
		})
	}
	merged := &pb.Node{Values: []int32{9}}
	v2.Merge(view(t, merged).Interface(), wrapped)
	if !bytes.Equal(merged.Data, original.Data) || len(merged.Values) != 3 {
		t.Fatal("merge")
	}
}
func mustPanic(t *testing.T, fn func()) {
	t.Helper()
	defer func() {
		if recover() == nil {
			t.Error("mutation accepted")
		}
	}()
	fn()
}
func TestNilReadonlyAndPresence(t *testing.T) {
	zero := view(t, (*pb.Node)(nil))
	if zero.IsValid() || zero.Has(field(zero, "count")) {
		t.Fatal("nil validity")
	}
	mustPanic(t, func() { zero.Set(field(zero, "count"), protoreflect.ValueOfInt32(1)) })
	m := view(t, new(pb.Node))
	for _, name := range []string{"values", "children"} {
		l := m.Get(field(m, name)).List()
		if l.IsValid() {
			t.Fatal("nil list valid")
		}
		mustPanic(t, func() { l.Append(l.NewElement()) })
	}
	mustPanic(t, func() { m.Get(field(m, "scores")).Map().Clear(protoreflect.ValueOfString("x").MapKey()) })
	n := &pb.Node{Data: make([]byte, 0), Ratio: math.Copysign(0, -1), Small: float32(math.Copysign(0, -1))}
	r := view(t, n)
	if r.Has(field(r, "data")) || !r.Has(field(r, "ratio")) || !r.Has(field(r, "small")) {
		t.Fatal("implicit presence")
	}
	for i := 0; i < r.Descriptor().Fields().Len(); i++ {
		fd := r.Descriptor().Fields().Get(i)
		a, b := r.NewField(fd), dynamicpb.NewMessage(r.Descriptor()).NewField(fd)
		if fd.IsList() {
			if a.List().NewElement().IsValid() != b.List().NewElement().IsValid() {
				t.Fatal(fd.FullName())
			}
		}
	}
}
func TestUnknownAndDescriptors(t *testing.T) {
	m := view(t, new(pb.Node))
	raw := []byte{0xa0, 6, 7}
	m.SetUnknown(raw)
	wire, e := v2.Marshal(m.Interface())
	if e != nil || !bytes.Equal(wire, raw) {
		t.Fatal(wire, e)
	}
	clone := v2.Clone(m.Interface())
	clone.ProtoReflect().SetUnknown(nil)
	if !bytes.Equal(m.GetUnknown(), raw) {
		t.Fatal("unknown aliases clone")
	}
	foreign := dynamicpb.NewMessage(m.Descriptor())
	foreign.Set(field(m, "count"), protoreflect.ValueOfInt32(31))
	m.Set(field(m, "child"), protoreflect.ValueOfMessage(foreign))
	if m.Get(field(m, "child")).Message().Get(field(m, "count")).Int() != 31 {
		t.Fatal("dynamic message conversion")
	}
	mustPanic(t, func() { m.SetUnknown([]byte{0}) })
}
func TestConcurrentReadonly(t *testing.T) {
	n := &pb.Node{Count: 1, Child: &pb.Node{Name: "x"}, Nodes: map[string]*pb.Node{"x": {Count: 2}}}
	m := view(t, n)
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				v2.Marshal(m.Interface())
				protojson.Marshal(m.Interface())
				live.Wrap(n)
			}
		}()
	}
	wg.Wait()
}

func TestConcurrentColdMetadata(t *testing.T) {
	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			p, err := live.Wrap(&pb.Cold{Child: &pb.Node{Count: 3}})
			if err != nil {
				t.Error(err)
				return
			}
			if _, err = v2.Marshal(p); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
}

func TestOfficialReflectionContract(t *testing.T) {
	t.Run("Node", func(t *testing.T) { prototest.Message{}.Test(t, view(t, new(pb.Node)).Type()) })
	p, err := live.Wrap(new(pb.Cold))
	if err != nil {
		t.Fatal(err)
	}
	t.Run("Cold", func(t *testing.T) { prototest.Message{}.Test(t, p.ProtoReflect().Type()) })
}
