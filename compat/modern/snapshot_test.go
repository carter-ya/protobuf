package modern

import (
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"

	ed23 "github.com/gogo/protobuf/compat/editions/edition2023"
	ed24 "github.com/gogo/protobuf/compat/editions/edition2024"
	ed26 "github.com/gogo/protobuf/compat/editions/edition2026"
	optional "github.com/gogo/protobuf/compat/optional/reflection"
	gogo "github.com/gogo/protobuf/proto"
	legacy "github.com/gogo/protobuf/proto/test_proto"
	"github.com/gogo/protobuf/types"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"
)

func registry(t *testing.T) *Registry {
	t.Helper()
	raw, err := os.ReadFile(os.Getenv("GOGO_MODERN_DESCRIPTOR"))
	if err != nil {
		t.Fatal(err)
	}
	// Prefer the actual generated runtime metadata for frozen WKT sources;
	// their gogo options differ from the compiler's source .proto options.
	set := new(descriptorpb.FileDescriptorSet)
	if err := proto.Unmarshal(raw, set); err != nil {
		t.Fatal(err)
	}
	for i, file := range set.File {
		if source := runtimeDescriptor(file.GetName()); len(source) != 0 {
			z, err := gzip.NewReader(bytes.NewReader(source))
			if err != nil {
				t.Fatal(err)
			}
			data, err := io.ReadAll(z)
			z.Close()
			if err != nil {
				t.Fatal(err)
			}
			runtime := new(descriptorpb.FileDescriptorProto)
			if err := proto.Unmarshal(data, runtime); err != nil {
				t.Fatal(err)
			}
			runtime.Name = proto.String(file.GetName())
			set.File[i] = runtime
		}
	}
	raw, err = proto.Marshal(set)
	if err != nil {
		t.Fatal(err)
	}
	r, err := NewRegistry(raw)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func TestSnapshots(t *testing.T) {
	r := registry(t)
	state := optional.State_STATE_ONE
	for _, m := range []gogo.Message{
		&types.Duration{Seconds: 5, Nanos: 7},
		new(optional.Sample),
		&optional.Sample{Value: gogo.Int32(0), Data: []byte{}, State: &state, Choice: &optional.Sample_Name{Name: "x"}},
		&ed23.Sample{Value: gogo.Int32(0), ExplicitBytes: []byte{}, Delimited: &ed23.Child{Value: gogo.Int32(7)}, Closed: ed23.Closed_CLOSED_ONE.Enum(), Open: ed23.Open(99).Enum()},
		&ed24.Sample{Value: gogo.Int32(0)},
		&ed26.Sample{Status: ed26.Status_STATUS_OK.Enum(), Values: map[string]ed26.Status{"x": ed26.Status_STATUS_EMPTY}},
	} {
		t.Run(gogo.MessageName(m), func(t *testing.T) {
			before := gogo.Clone(m)
			snapshot, err := r.Snapshot(m)
			if err != nil {
				t.Fatal(err)
			}
			if err := r.Apply(m, snapshot); err != nil || !gogo.Equal(m, before) {
				t.Fatalf("roundtrip: %v", err)
			}
			unknown := []byte{0xc0, 0x3e, 7} // field 1000
			snapshot.SetUnknown(unknown)
			if !gogo.Equal(m, before) {
				t.Fatal("snapshot aliases original")
			}
			if err := r.Apply(m, snapshot); err != nil {
				t.Fatal(err)
			}
			back, err := r.Snapshot(m)
			if err != nil || !bytes.Equal(back.GetUnknown(), unknown) {
				t.Fatalf("unknown lost: %v", err)
			}
			unknown[2] = 8
			if bytes.Equal(back.GetUnknown(), unknown) {
				t.Fatal("Apply retained snapshot storage")
			}
		})
	}
}

func TestReflectionPresenceAndMutation(t *testing.T) {
	r := registry(t)
	original := &optional.Sample{Value: gogo.Int32(0), Data: []byte{}}
	m, err := r.Snapshot(original)
	if err != nil {
		t.Fatal(err)
	}
	fields := m.Descriptor().Fields()
	value, data, enabled := fields.ByName("value"), fields.ByName("data"), fields.ByName("enabled")
	if !m.Has(value) || !m.Has(data) || m.Has(enabled) || !value.ContainingOneof().IsSynthetic() {
		t.Fatal("optional metadata/presence lost")
	}
	m.Clear(value)
	m.Set(enabled, protoreflect.ValueOfBool(false))
	m.Set(data, protoreflect.ValueOfBytes([]byte{7}))
	if original.Value == nil || original.Enabled != nil || len(original.Data) != 0 {
		t.Fatal("reflection is not detached")
	}
	if err := r.Apply(original, m); err != nil {
		t.Fatal(err)
	}
	if original.Value != nil || original.Enabled == nil || *original.Enabled || !bytes.Equal(original.Data, []byte{7}) {
		t.Fatal("reflection mutation lost")
	}
	if _, err := protoregistry.GlobalTypes.FindMessageByName(m.Descriptor().FullName()); !errors.Is(err, protoregistry.NotFound) {
		t.Fatal("experiment polluted V2 global registry")
	}
}

func TestJSONAnyAndCustomEnumNames(t *testing.T) {
	r := registry(t)
	value := &optional.Sample{Value: gogo.Int32(0), Data: []byte{}}
	packed, err := types.MarshalAny(value)
	if err != nil {
		t.Fatal(err)
	}
	for _, original := range []gogo.Message{packed, &ed26.Sample{Status: ed26.Status_STATUS_OK.Enum()}} {
		m, err := r.Snapshot(original)
		if err != nil {
			t.Fatal(err)
		}
		text, err := (protojson.MarshalOptions{Resolver: r.Types}).Marshal(m)
		if err != nil {
			t.Fatal(err)
		}
		if _, ok := original.(*ed26.Sample); ok && !bytes.Contains(text, []byte(`ok\"line\n雪`)) {
			t.Fatalf("2026 name lost: %s", text)
		}
		parsed := dynamicpb.NewMessage(m.Descriptor())
		if err := (protojson.UnmarshalOptions{Resolver: r.Types}).Unmarshal(text, parsed); err != nil {
			t.Fatal(err)
		}
		if !proto.Equal(m, parsed) {
			t.Fatalf("JSON changed semantics: %s", text)
		}
		if err := r.Apply(original, parsed); err != nil {
			t.Fatal(err)
		}
	}
	m, _ := r.Snapshot(packed)
	m.Set(m.Descriptor().Fields().ByName("type_url"), protoreflect.ValueOfString("type.googleapis.com/unknown.Payload"))
	if _, err := (protojson.MarshalOptions{Resolver: r.Types}).Marshal(m); err == nil {
		t.Fatal("unknown Any payload accepted")
	}
}

func TestExtensionsAndClosedEnums(t *testing.T) {
	r := registry(t)
	original := new(ed23.Extensible)
	if err := gogo.SetExtension(original, ed23.E_ExtClosed, ed23.Closed_CLOSED_ONE.Enum()); err != nil {
		t.Fatal(err)
	}
	if err := gogo.SetExtension(original, ed23.E_Declaring_NestedExt, gogo.Int32(0)); err != nil {
		t.Fatal(err)
	}
	m, err := r.Snapshot(original)
	if err != nil {
		t.Fatal(err)
	}
	x, err := r.Types.FindExtensionByName("compat.editions.v2023.ext_closed")
	if err != nil {
		t.Fatal(err)
	}
	if proto.GetExtension(m, x).(protoreflect.EnumNumber) != 1 {
		t.Fatal("extension lost")
	}
	proto.SetExtension(m, x, protoreflect.EnumNumber(0))
	if err := r.Apply(original, m); err != nil {
		t.Fatal(err)
	}
	actual, err := gogo.GetExtension(original, ed23.E_ExtClosed)
	if err != nil || *actual.(*ed23.Closed) != 0 {
		t.Fatalf("extension update: %v", err)
	}
	// Unknown values in a closed enum remain unknown on the gogo side.
	wire := []byte{0x48, 99}
	var closed ed23.Sample
	if err := gogo.Unmarshal(wire, &closed); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Snapshot(&closed); !errors.Is(err, ErrUnsupportedSemantics) {
		t.Fatalf("closed-enum semantic mismatch must be rejected: %v", err)
	}
	if closed.Closed != nil || !bytes.Equal(closed.XXX_unrecognized, wire) {
		t.Fatal("closed unknown changed")
	}

}

func TestRejectionIsTransactional(t *testing.T) {
	r := registry(t)
	if _, err := r.Snapshot((*optional.Sample)(nil)); err == nil {
		t.Fatal("nil accepted")
	}
	if _, err := r.Snapshot(new(ed23.Required)); err == nil {
		t.Fatal("missing required accepted")
	}
	dst := &optional.Sample{Value: gogo.Int32(7)}
	before := gogo.Clone(dst)
	wrong, _ := r.Snapshot(new(ed24.Sample))
	if err := r.Apply(dst, wrong); err == nil || !gogo.Equal(dst, before) {
		t.Fatal("mismatched descriptor mutated target")
	}
	if _, err := NewRegistry([]byte{255}); err == nil {
		t.Fatal("malformed descriptor accepted")
	}
	set := &descriptorpb.FileDescriptorSet{File: []*descriptorpb.FileDescriptorProto{{Name: proto.String("missing.proto"), Dependency: []string{"absent.proto"}}}}
	wire, _ := proto.Marshal(set)
	if _, err := NewRegistry(wire); err == nil {
		t.Fatal("missing dependency accepted")
	}
}

func TestLegacyMessageSetRejected(t *testing.T) {
	// Modern Go rejects MessageSet by default. Preserve that error rather
	// than silently rewriting descriptors to make the conversion appear safe.
	raw, _ := new(legacy.OldMessage).Descriptor()
	z, err := gzip.NewReader(bytes.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	data, err := io.ReadAll(z)
	z.Close()
	if err != nil {
		t.Fatal(err)
	}
	file := new(descriptorpb.FileDescriptorProto)
	if err := proto.Unmarshal(data, file); err != nil {
		t.Fatal(err)
	}
	set, _ := proto.Marshal(&descriptorpb.FileDescriptorSet{File: []*descriptorpb.FileDescriptorProto{file}})
	if _, err := NewRegistry(set); err == nil || !strings.Contains(err.Error(), "MessageSet") {
		t.Fatalf("expected explicit MessageSet rejection, got %v", err)
	}
}

type noUnknown struct {
	Value *int32 `protobuf:"varint,1,opt,name=value"`
}

func (m *noUnknown) Reset()       { *m = noUnknown{} }
func (*noUnknown) String() string { return "noUnknown" }
func (*noUnknown) ProtoMessage()  {}
func (*noUnknown) Descriptor() ([]byte, []int) {
	var compressed bytes.Buffer
	z := gzip.NewWriter(&compressed)
	wire, _ := proto.Marshal(lossDescriptor())
	z.Write(wire)
	z.Close()
	return compressed.Bytes(), []int{0}
}
func init() { gogo.RegisterType((*noUnknown)(nil), "experiment.NoUnknown") }

func TestLossyConversionRejected(t *testing.T) {
	file := lossDescriptor()
	wire, _ := proto.Marshal(&descriptorpb.FileDescriptorSet{File: []*descriptorpb.FileDescriptorProto{file}})
	r, err := NewRegistry(wire)
	if err != nil {
		t.Fatal(err)
	}
	dst := &noUnknown{Value: gogo.Int32(7)}
	m, err := r.Snapshot(dst)
	if err != nil {
		t.Fatal(err)
	}
	m.Set(m.Descriptor().Fields().Get(0), protoreflect.ValueOfInt32(8))
	m.SetUnknown([]byte{16, 1})
	if err := r.Apply(dst, m); !errors.Is(err, ErrLossyConversion) || *dst.Value != 7 {
		t.Fatalf("lossy conversion mutated target: %v", err)
	}
}

func lossDescriptor() *descriptorpb.FileDescriptorProto {
	return &descriptorpb.FileDescriptorProto{Name: proto.String("loss.proto"), Package: proto.String("experiment"), MessageType: []*descriptorpb.DescriptorProto{{Name: proto.String("NoUnknown"), Field: []*descriptorpb.FieldDescriptorProto{{Name: proto.String("value"), Number: proto.Int32(1), Type: descriptorpb.FieldDescriptorProto_TYPE_INT32.Enum(), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum()}}}}}
}

func TestMismatchedSourceSchemaRejected(t *testing.T) {
	file := lossDescriptor()
	file.MessageType[0].Field[0].Type = descriptorpb.FieldDescriptorProto_TYPE_BOOL.Enum()
	wire, _ := proto.Marshal(&descriptorpb.FileDescriptorSet{File: []*descriptorpb.FileDescriptorProto{file}})
	r, err := NewRegistry(wire)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := r.Snapshot(&noUnknown{Value: gogo.Int32(7)}); err == nil {
		t.Fatal("same-name, different schema accepted")
	}
}

func TestClosedEnumAndCycleGuards(t *testing.T) {
	r := registry(t)
	for _, wire := range [][]byte{
		{0x48, 99},                    // singular
		{0x52, 1, 99},                 // packed list
		{0x5a, 5, 10, 1, 'x', 16, 99}, // map value
		{0x80, 1, 99},                 // oneof
	} {
		closed := new(ed23.Sample)
		if err := gogo.Unmarshal(wire, closed); err != nil {
			t.Fatal(err)
		}
		if _, err := r.Snapshot(closed); !errors.Is(err, ErrUnsupportedSemantics) {
			t.Fatalf("closed enum %x not rejected: %v", wire, err)
		}
		packed, err := types.MarshalAny(closed)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := r.Snapshot(packed); !errors.Is(err, ErrUnsupportedSemantics) {
			t.Fatalf("Any hid closed enum %x: %v", wire, err)
		}
	}
	extended := new(ed23.Extensible)
	if err := gogo.Unmarshal([]byte{0xa0, 6, 99}, extended); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Snapshot(extended); !errors.Is(err, ErrUnsupportedSemantics) {
		t.Fatalf("closed extension not rejected: %v", err)
	}
	file := lossDescriptor()
	file.MessageType[0].Field = append(file.MessageType[0].Field, &descriptorpb.FieldDescriptorProto{Name: proto.String("child"), Number: proto.Int32(2), Type: descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(), TypeName: proto.String(".experiment.NoUnknown"), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum()})
	wire, _ := proto.Marshal(&descriptorpb.FileDescriptorSet{File: []*descriptorpb.FileDescriptorProto{file}})
	r, err := NewRegistry(wire)
	if err != nil {
		t.Fatal(err)
	}
	typeOf, _ := r.Types.FindMessageByName("experiment.NoUnknown")
	cycle := typeOf.New()
	cycle.Set(cycle.Descriptor().Fields().ByName("child"), protoreflect.ValueOfMessage(cycle))
	if err := r.validateMessage(cycle); !errors.Is(err, ErrUnsupportedSemantics) {
		t.Fatalf("cycle not rejected: %v", err)
	}
}

func TestConcurrentRegistryReads(t *testing.T) {
	r := registry(t)
	for i := 0; i < 8; i++ {
		t.Run(fmt.Sprintf("reader-%d", i), func(t *testing.T) {
			t.Parallel()
			for j := 0; j < 20; j++ {
				m, err := r.Snapshot(&optional.Sample{Value: gogo.Int32(int32(j)), Data: []byte{}})
				if err != nil {
					t.Fatal(err)
				}
				var result optional.Sample
				if err := r.Apply(&result, m); err != nil || result.Value == nil || *result.Value != int32(j) || result.Data == nil {
					t.Fatalf("concurrent conversion: %v", err)
				}
			}
		})
	}
}

func TestImportedSourceMismatchRejected(t *testing.T) {
	for _, transitive := range []bool{false, true} {
		good := registry(t)
		set := new(descriptorpb.FileDescriptorSet)
		for name, file := range good.sourceFiles {
			copy := proto.Clone(file).(*descriptorpb.FileDescriptorProto)
			if !transitive && name == "compat/editions/edition2023/imported.proto" {
				copy.EnumType[0].Value[0].Name = proto.String("RENAMED_FIRST")
			}
			if transitive && name == "google/protobuf/descriptor.proto" {
				copy.Options.GoPackage = proto.String("example.com/changed/descriptor")
			}
			set.File = append(set.File, copy)
		}
		raw, _ := proto.Marshal(set)
		altered, err := NewRegistry(raw)
		if err != nil {
			t.Fatal(err)
		}
		var original gogo.Message = &ed23.Sample{Imported: ed23.Imported_FIRST.Enum()}
		if transitive {
			original = &optional.Sample{Value: gogo.Int32(7)}
		}
		if snapshot, err := altered.Snapshot(original); err == nil {
			text, _ := (protojson.MarshalOptions{Resolver: altered.Types}).Marshal(snapshot)
			t.Fatalf("accepted changed dependency (transitive=%v): %s", transitive, text)
		}
		name, err := good.sourceName(original)
		if err != nil {
			t.Fatal(err)
		}
		typeOf, _ := altered.Types.FindMessageByName(name)
		snapshot := dynamicpb.NewMessage(typeOf.Descriptor())
		wire, _ := gogo.Marshal(original)
		if err := (proto.UnmarshalOptions{Resolver: altered.Types}).Unmarshal(wire, snapshot); err != nil {
			t.Fatal(err)
		}
		before := gogo.Clone(original)
		if err := altered.Apply(original, snapshot); err == nil || !gogo.Equal(before, original) {
			t.Fatalf("Apply accepted changed dependency (transitive=%v): %v", transitive, err)
		}
	}
}
