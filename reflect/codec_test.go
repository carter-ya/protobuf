package gogoreflect_test

import (
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"

	gogo "github.com/gogo/protobuf/proto"
	live "github.com/gogo/protobuf/reflect"
	pb "github.com/gogo/protobuf/reflect/internal/testpb"
	"github.com/gogo/protobuf/types"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/encoding/protowire"
	v2 "google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/testing/prototest"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/known/anypb"
)

func closedView(t *testing.T, n *pb.Closed) protoreflect.Message {
	t.Helper()
	p, e := live.Wrap(n)
	if e != nil {
		t.Fatal(e)
	}
	return p.ProtoReflect()
}
func varint(n protowire.Number, v uint64) []byte {
	return protowire.AppendVarint(protowire.AppendTag(nil, n, protowire.VarintType), v)
}
func delimited(n protowire.Number, b []byte) []byte {
	return protowire.AppendBytes(protowire.AppendTag(nil, n, protowire.BytesType), b)
}
func TestClosedEnumWireAndDiscard(t *testing.T) {
	unknown := varint(1, 99)
	wire := append(append([]byte(nil), unknown...), varint(1, 7)...)
	n := new(pb.Closed)
	m := closedView(t, n)
	if err := v2.Unmarshal(wire, m.Interface()); err != nil {
		t.Fatal(err)
	}
	if n.GetStatus() != pb.Status_BAD || !bytes.Equal(m.GetUnknown(), unknown) {
		t.Fatal(n, m.GetUnknown())
	}
	old := varint(200, 3)
	m.SetUnknown(old)
	if err := (v2.UnmarshalOptions{Merge: true, DiscardUnknown: true}).Unmarshal(unknown, m.Interface()); err != nil {
		t.Fatal(err)
	}
	if n.GetStatus() != pb.Status_BAD || !bytes.Equal(m.GetUnknown(), old) {
		t.Fatal("discard erased existing unknowns", n, m.GetUnknown())
	}
	n = new(pb.Closed)
	m = closedView(t, n)
	packed := delimited(2, []byte{5, 99, 7})
	if err := v2.Unmarshal(packed, m.Interface()); err != nil {
		t.Fatal(err)
	}
	if len(n.List) != 2 || n.List[0] != 5 || n.List[1] != 7 || !bytes.Equal(m.GetUnknown(), varint(2, 99)) {
		t.Fatal(n, m.GetUnknown())
	}
	// A rejected closed-enum oneof occurrence must not replace the current member.
	n.Choice = &pb.Closed_Number{Number: 4}
	if err := (v2.UnmarshalOptions{Merge: true}).Unmarshal(varint(4, 99), m.Interface()); err != nil || n.GetNumber() != 4 {
		t.Fatal(err, n)
	}
	order := append(append(varint(4, 5), varint(5, 8)...), varint(4, 7)...)
	if err := v2.Unmarshal(order, m.Interface()); err != nil || n.GetSelected() != 7 {
		t.Fatal("oneof wire order", err, n)
	}
	// Invalid closed map values preserve the complete entry as unknown wire.
	entry := append(delimited(1, []byte("x")), varint(2, 99)...)
	mapwire := delimited(3, entry)
	if err := v2.Unmarshal(mapwire, m.Interface()); err != nil || len(n.Flags) != 0 || !bytes.Equal(m.GetUnknown(), mapwire) {
		t.Fatal("map enum", err, n, m.GetUnknown())
	}
}
func TestFinalRequiredAndRecursion(t *testing.T) {
	n := new(pb.Closed)
	m := closedView(t, n)
	wire := append(delimited(6, nil), delimited(6, varint(1, 7))...)
	if err := v2.Unmarshal(wire, m.Interface()); err != nil || n.Child.GetValue() != 7 {
		t.Fatal("final required merge", err, n)
	}
	if err := v2.Unmarshal(delimited(6, nil), m.Interface()); err == nil {
		t.Fatal("missing required accepted")
	}
	if err := (v2.UnmarshalOptions{AllowPartial: true}).Unmarshal(delimited(6, nil), m.Interface()); err != nil {
		t.Fatal("partial rejected", err)
	}
	if _, err := (v2.MarshalOptions{AllowPartial: true}).Marshal(m.Interface()); err != nil {
		t.Fatal("partial marshal", err)
	}
	if _, err := v2.Marshal(m.Interface()); err == nil {
		t.Fatal("required marshal accepted")
	}
	node := view(t, new(pb.Node))
	if err := (v2.UnmarshalOptions{RecursionLimit: 1}).Unmarshal(delimited(5, varint(1, 7)), node.Interface()); err == nil {
		t.Fatal("recursion limit ignored")
	}
	if err := (v2.UnmarshalOptions{RecursionLimit: 2}).Unmarshal(delimited(5, varint(1, 7)), node.Interface()); err != nil {
		t.Fatal(err)
	}
}
func TestLiveExtensionsAndResolver(t *testing.T) {
	n := new(pb.Closed)
	m := closedView(t, n)
	r, err := live.NewRegistry(n)
	if err != nil {
		t.Fatal(err)
	}
	scalar, err := r.FindExtensionByNumber(m.Descriptor().FullName(), 104)
	if err != nil {
		t.Fatal(err)
	}
	if v := v2.GetExtension(m.Interface(), scalar).(int32); v != 42 {
		t.Fatal("default", v)
	}
	v2.SetExtension(m.Interface(), scalar, int32(9))
	got, e := gogo.GetExtension(n, pb.E_ScalarExt)
	if e != nil || *got.(*int32) != 9 {
		t.Fatal(e, got)
	}
	list, _ := r.FindExtensionByNumber(m.Descriptor().FullName(), 101)
	l := m.Mutable(list.TypeDescriptor()).List()
	l.Append(protoreflect.ValueOfEnum(5))
	l.Append(protoreflect.ValueOfEnum(7))
	got, e = gogo.GetExtension(n, pb.E_ListExt)
	if e != nil || len(got.([]pb.Status)) != 2 {
		t.Fatal("live extension list", e, got)
	}
	msg, _ := r.FindExtensionByNumber(m.Descriptor().FullName(), 102)
	child := m.Mutable(msg.TypeDescriptor()).Message()
	child.Set(field(child, "count"), protoreflect.ValueOfInt32(13))
	got, e = gogo.GetExtension(n, pb.E_MsgExt)
	if e != nil || got.(*pb.Node).Count != 13 {
		t.Fatal("live extension message", e, got)
	}
	wire, err := v2.Marshal(m.Interface())
	if err != nil {
		t.Fatal(err)
	}
	back := new(pb.Closed)
	if err = gogo.Unmarshal(wire, back); err != nil {
		t.Fatal(err)
	}
	if !gogo.Equal(n, back) {
		t.Fatal("cross decode", n, back)
	}
	clone := v2.Clone(m.Interface())
	if !v2.Equal(clone, m.Interface()) {
		t.Fatal("extension clone")
	}
	raw := varint(104, 21)
	empty, _ := live.NewRegistry()
	other := new(pb.Closed)
	o := closedView(t, other)
	if err = (v2.UnmarshalOptions{Resolver: empty}).Unmarshal(raw, o.Interface()); err != nil {
		t.Fatal(err)
	}
	if m := o; m.Has(scalar.TypeDescriptor()) || !bytes.Equal(m.GetUnknown(), raw) {
		t.Fatal("explicit empty resolver ignored", m.GetUnknown())
	}
	if err = v2.Unmarshal(raw, o.Interface()); err != nil || !o.Has(scalar.TypeDescriptor()) {
		t.Fatal("default native resolution", err)
	}
	// Registered closed enum extensions use the native decoder, not Go's open treatment.
	enumraw := varint(100, 99)
	if err = v2.Unmarshal(enumraw, o.Interface()); err != nil {
		t.Fatal(err)
	}
	if o.Has(mustExtension(t, r, 100).TypeDescriptor()) || !bytes.Equal(o.GetUnknown(), enumraw) {
		t.Fatal("closed enum extension promoted", other, o.GetUnknown())
	}
	prototest.Message{Resolver: r}.Test(t, m.Type())
}
func mustExtension(t *testing.T, r *live.Registry, n protoreflect.FieldNumber) protoreflect.ExtensionType {
	t.Helper()
	x, e := r.FindExtensionByNumber("live.Closed", n)
	if e != nil {
		t.Fatal(e)
	}
	return x
}
func TestAnyLocalRegistry(t *testing.T) {
	n := &pb.Node{Count: 17, Name: "payload"}
	m := view(t, n)
	r, e := live.NewRegistry(n, new(types.Any))
	if e != nil {
		t.Fatal(e)
	}
	a, e := anypb.New(m.Interface())
	if e != nil {
		t.Fatal(e)
	}
	unpacked, e := anypb.UnmarshalNew(a, v2.UnmarshalOptions{Resolver: r})
	if e != nil || !v2.Equal(m.Interface(), unpacked) {
		t.Fatal(e)
	}
	nativeAny := &types.Any{TypeUrl: a.TypeUrl, Value: a.Value}
	p, e := live.Wrap(nativeAny)
	if e != nil {
		t.Fatal(e)
	}
	b, e := (protojson.MarshalOptions{Resolver: r}).Marshal(p)
	if e != nil {
		t.Fatal(e)
	}
	dest := new(types.Any)
	d, e := live.Wrap(dest)
	if e != nil {
		t.Fatal(e)
	}
	if e = (protojson.UnmarshalOptions{Resolver: r}).Unmarshal(b, d); e != nil {
		t.Fatal(e)
	}
	decoded := new(pb.Node)
	if e = gogo.Unmarshal(dest.Value, decoded); e != nil || !gogo.Equal(n, decoded) {
		t.Fatal(e, decoded)
	}
}

func TestUnsupportedRepresentationsAreDiagnosed(t *testing.T) {
	cases := []struct {
		m    gogo.Message
		want string
	}{
		{new(pb.BytesExtensions), "map-backed extensions"},
		{new(pb.NoUnknown), "retain unknown"},
		{new(pb.Flattened), "pointer message storage"},
		{new(pb.StdMapping), "custom/std mapping"},
		{new(pb.CustomMapping), "custom/std mapping"},
	}
	for _, c := range cases {
		if _, err := live.Wrap(c.m); err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%T: %v; want %s", c.m, err, c.want)
		}
	}
}
func TestUnknownExtensionStorageAndSnapshots(t *testing.T) {
	n := new(pb.Closed)
	m := closedView(t, n)
	raw := append(varint(108, 7), varint(107, 9)...)
	m.SetUnknown(raw)
	expected := append(varint(107, 9), varint(108, 7)...)
	for i := 0; i < 30; i++ {
		if !bytes.Equal(m.GetUnknown(), expected) {
			t.Fatal("unknown extension view unstable", m.GetUnknown())
		}
	}
	snapshot, err := gogo.UnregisteredExtensionWire(n)
	if err != nil {
		t.Fatal(err)
	}
	snapshot[107][len(snapshot[107])-1] = 99
	if !bytes.Equal(m.GetUnknown(), expected) {
		t.Fatal("extension snapshot aliases native bytes")
	}
	// Replacing unknowns must leave registered fields untouched.
	if err := gogo.SetExtension(n, pb.E_ScalarExt, gogo.Int32(31)); err != nil {
		t.Fatal(err)
	}
	m.SetUnknown(nil)
	if got, err := gogo.GetExtension(n, pb.E_ScalarExt); err != nil || *got.(*int32) != 31 {
		t.Fatal(err, got)
	}
	if unknown, err := gogo.UnregisteredExtensionWire(n); err != nil || len(unknown) != 0 {
		t.Fatal(err, unknown)
	}
}
func TestConcurrentExtensionReads(t *testing.T) {
	n := new(pb.Closed)
	r, err := live.NewRegistry(n)
	if err != nil {
		t.Fatal(err)
	}
	if err := gogo.Unmarshal(append(varint(104, 7), delimited(102, varint(1, 9))...), n); err != nil {
		t.Fatal(err)
	}
	m := closedView(t, n)
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				if _, e := v2.Marshal(m.Interface()); e != nil {
					t.Error(e)
				}
				if _, e := (protojson.MarshalOptions{Resolver: r}).Marshal(m.Interface()); e != nil {
					t.Error(e)
				}
				live.Wrap(n)
			}
		}()
	}
	wg.Wait()
}

func TestLateExtensionRegistration(t *testing.T) {
	if gogo.RegisteredExtensions(new(pb.Closed))[109] != nil {
		t.Skip("serial late registration already exercised")
	}
	n := new(pb.Closed)
	m := closedView(t, n)
	r, err := live.NewRegistry(n)
	if err != nil {
		t.Fatal(err)
	}
	raw := varint(109, 17)
	if err = v2.Unmarshal(raw, m.Interface()); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(m.GetUnknown(), raw) {
		t.Fatal("unregistered wire lost")
	}
	file := &descriptorpb.FileDescriptorProto{Name: v2.String("live-late.proto"), Package: v2.String("live"), Syntax: v2.String("proto2"), Dependency: []string{"reflect/internal/testpb/closed.proto"}, Extension: []*descriptorpb.FieldDescriptorProto{{Name: v2.String("late_scalar"), Number: v2.Int32(109), Extendee: v2.String(".live.Closed"), Type: descriptorpb.FieldDescriptorProto_TYPE_INT32.Enum(), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum()}}}
	data, err := v2.Marshal(file)
	if err != nil {
		t.Fatal(err)
	}
	var compressed bytes.Buffer
	z := gzip.NewWriter(&compressed)
	z.Write(data)
	z.Close()
	gogo.RegisterFile("live-late.proto", compressed.Bytes())
	native := &gogo.ExtensionDesc{ExtendedType: (*pb.Closed)(nil), ExtensionType: (*int32)(nil), Field: 109, Name: "live.late_scalar", Filename: "live-late.proto", Tag: "varint,109,opt,name=late_scalar"}
	gogo.RegisterExtension(native)
	x, err := r.FindExtensionByNumber("live.Closed", 109)
	if err != nil {
		t.Fatal(err)
	}
	if !m.Has(x.TypeDescriptor()) || v2.GetExtension(m.Interface(), x).(int32) != 17 || len(m.GetUnknown()) != 0 {
		t.Fatal("late binding not visible")
	}
	v2.SetExtension(m.Interface(), x, int32(21))
	v, err := gogo.GetExtension(n, native)
	if err != nil || *v.(*int32) != 21 {
		t.Fatal(err, v)
	}
}

func TestGroupAndNestedDiscard(t *testing.T) {
	native := new(pb.GroupContainer)
	wrapped, err := live.Wrap(native)
	if err != nil {
		t.Fatal(err)
	}
	wire := []byte{0x0b, 0x10, 7, 0x0c}
	if err = v2.Unmarshal(wire, wrapped); err != nil || native.GetLegacy().GetValue() != 7 {
		t.Fatal("group decode", err, native)
	}
	encoded, err := v2.Marshal(wrapped)
	if err != nil || !bytes.Equal(encoded, wire) {
		t.Fatal("group encode", encoded, err)
	}
	prototest.Message{}.Test(t, wrapped.ProtoReflect().Type())
	child := &pb.Node{Name: "kept"}
	root := &pb.Node{Child: child}
	m := view(t, root)
	old := varint(200, 7)
	view(t, child).SetUnknown(old)
	input := delimited(5, append(varint(1, 9), varint(201, 11)...))
	if err = (v2.UnmarshalOptions{Merge: true, DiscardUnknown: true}).Unmarshal(input, m.Interface()); err != nil {
		t.Fatal(err)
	}
	if root.Child != child || child.Count != 9 || !bytes.Equal(view(t, child).GetUnknown(), old) {
		t.Fatal("nested discard changed old unknowns", root)
	}
}

type rejectingResolver struct{ err error }

func (r rejectingResolver) FindExtensionByName(protoreflect.FullName) (protoreflect.ExtensionType, error) {
	return nil, r.err
}
func (r rejectingResolver) FindExtensionByNumber(protoreflect.FullName, protoreflect.FieldNumber) (protoreflect.ExtensionType, error) {
	return nil, r.err
}
func TestResolverErrorsAndMalformedClosedWire(t *testing.T) {
	n := new(pb.Closed)
	m := closedView(t, n)
	sentinel := errors.New("resolver rejected lookup")
	if err := (v2.UnmarshalOptions{Resolver: rejectingResolver{sentinel}}).Unmarshal(varint(104, 7), m.Interface()); !errors.Is(err, sentinel) {
		t.Fatal("resolver error lost", err)
	}
	if err := v2.Unmarshal([]byte{0x12, 1, 0x80}, m.Interface()); err == nil {
		t.Fatal("malformed packed closed enum accepted")
	}
	empty, _ := live.NewRegistry()
	unknown := varint(100, 99)
	if err := (v2.UnmarshalOptions{Resolver: empty}).Unmarshal(unknown, m.Interface()); err != nil || m.Has(mustExtension(t, mustRegistry(t, n), 100).TypeDescriptor()) || !bytes.Equal(m.GetUnknown(), unknown) {
		t.Fatal("closed extension resolver ignored", err, n, m.GetUnknown())
	}
}
func mustRegistry(t *testing.T, n gogo.Message) *live.Registry {
	t.Helper()
	r, e := live.NewRegistry(n)
	if e != nil {
		t.Fatal(e)
	}
	return r
}

func TestNestedExtensionResolvers(t *testing.T) {
	for _, explicit := range []bool{false, true} {
		t.Run(fmt.Sprint(explicit), func(t *testing.T) {
			n := new(pb.Envelope)
			wrapped, e := live.Wrap(n)
			if e != nil {
				t.Fatal(e)
			}
			opts := v2.UnmarshalOptions{}
			if explicit {
				r, e := live.NewRegistry(n)
				if e != nil {
					t.Fatal(e)
				}
				opts.Resolver = r
			}
			entry := append(delimited(1, []byte("x")), delimited(2, varint(104, 11))...)
			wire := append(append(delimited(1, varint(104, 7)), delimited(2, varint(104, 9))...), delimited(3, entry)...)
			if e = opts.Unmarshal(wire, wrapped); e != nil {
				t.Fatal(e)
			}
			for i, m := range []*pb.Closed{n.Child, n.Children[0], n.Nodes["x"]} {
				got, e := gogo.GetExtension(m, pb.E_ScalarExt)
				if e != nil || *got.(*int32) != int32(7+2*i) {
					t.Fatal("nested extension", i, e, got)
				}
			}
		})
	}
}
func TestMergeDecodedClosedExtensions(t *testing.T) {
	n := new(pb.Closed)
	m := closedView(t, n)
	r := mustRegistry(t, n)
	status := mustExtension(t, r, 100)
	list := mustExtension(t, r, 101)
	v2.SetExtension(m.Interface(), status, pb.Status_GOOD)
	if e := (v2.UnmarshalOptions{Merge: true}).Unmarshal(varint(100, 7), m.Interface()); e != nil {
		t.Fatal(e)
	}
	if v := v2.GetExtension(m.Interface(), status).(pb.Status); v != pb.Status_BAD {
		t.Fatal("decoded singular update lost", v)
	}
	v2.SetExtension(m.Interface(), list, []pb.Status{pb.Status_GOOD})
	if e := (v2.UnmarshalOptions{Merge: true}).Unmarshal(delimited(101, []byte{7}), m.Interface()); e != nil {
		t.Fatal(e)
	}
	got := v2.GetExtension(m.Interface(), list).([]pb.Status)
	if len(got) != 2 || got[0] != 5 || got[1] != 7 {
		t.Fatal("decoded repeated update lost", got)
	}
	// Materialized-by-read storage must also see subsequent valid/unknown values.
	if e := (v2.UnmarshalOptions{Merge: true}).Unmarshal(append(varint(100, 5), varint(100, 99)...), m.Interface()); e != nil {
		t.Fatal(e)
	}
	if v := v2.GetExtension(m.Interface(), status).(pb.Status); v != pb.Status_GOOD {
		t.Fatal("last valid enum lost", v)
	}
	if !bytes.Equal(m.GetUnknown(), varint(100, 99)) {
		t.Fatal("unknown closed extension lost", m.GetUnknown())
	}
}
