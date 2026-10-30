package optional_test

import (
	"bytes"
	"compress/gzip"
	"io/ioutil"
	"reflect"
	"strings"
	"testing"

	"github.com/gogo/protobuf/compat/optional/fast"
	"github.com/gogo/protobuf/compat/optional/reflection"
	"github.com/gogo/protobuf/jsonpb"
	"github.com/gogo/protobuf/proto"
	descriptor "github.com/gogo/protobuf/protoc-gen-gogo/descriptor"
)

func samples() []proto.Message { return []proto.Message{&reflection.Sample{}, &fast.Sample{}} }
func fresh(m proto.Message) proto.Message {
	return reflect.New(reflect.TypeOf(m).Elem()).Interface().(proto.Message)
}
func field(m proto.Message, name string) reflect.Value {
	return reflect.ValueOf(m).Elem().FieldByName(name)
}

func TestZeroValuePresence(t *testing.T) {
	for _, m := range samples() {
		t.Run(reflect.TypeOf(m).String(), func(t *testing.T) {
			if field(m, "Value").Kind() != reflect.Ptr || field(m, "Choice").Kind() != reflect.Interface {
				t.Fatal("optional changed real oneof or scalar API")
			}
			field(m, "Value").Set(reflect.ValueOf(proto.Int32(0)))
			b, err := proto.Marshal(m)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(b, []byte{8, 0}) {
				t.Fatalf("explicit zero lost: %x", b)
			}
			n := fresh(m)
			if err := proto.Unmarshal(b, n); err != nil {
				t.Fatal(err)
			}
			if field(n, "Value").IsNil() {
				t.Fatal("zero decoded without presence")
			}
			if proto.Equal(m, fresh(m)) {
				t.Fatal("unset equals explicitly set zero")
			}
			if !proto.Equal(m, proto.Clone(m)) {
				t.Fatal("Clone lost presence")
			}
			dst := fresh(m)
			field(dst, "Value").Set(reflect.ValueOf(proto.Int32(7)))
			proto.Merge(dst, m)
			if field(dst, "Value").Elem().Int() != 0 {
				t.Fatal("Merge ignored explicit zero")
			}
			j, err := (&jsonpb.Marshaler{}).MarshalToString(m)
			if err != nil {
				t.Fatal(err)
			}
			if j != "{\"value\":0}" {
				t.Fatalf("JSON lost presence: %s", j)
			}
			j, err = (&jsonpb.Marshaler{EmitDefaults: true}).MarshalToString(fresh(m))
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(j, "\"value\"") {
				t.Fatalf("EmitDefaults invented optional presence: %s", j)
			}
		})
	}
}

func TestEveryScalarAndEmptyBytes(t *testing.T) {
	for _, m := range samples() {
		v := reflect.ValueOf(m).Elem()
		for i := 0; i < v.NumField(); i++ {
			f := v.Field(i)
			if f.Kind() == reflect.Ptr {
				f.Set(reflect.New(f.Type().Elem()))
			}
		}
		field(m, "Data").SetBytes([]byte{})
		b, err := proto.Marshal(m)
		if err != nil {
			t.Fatal(err)
		}
		if proto.Size(m) != len(b) {
			t.Fatalf("%T Size mismatch", m)
		}
		n := fresh(m)
		if err := proto.Unmarshal(b, n); err != nil {
			t.Fatal(err)
		}
		if !proto.Equal(m, n) {
			t.Fatalf("%T scalar zero presence lost", m)
		}
		if field(n, "Data").IsNil() {
			t.Fatal("empty bytes became nil")
		}
		if !proto.Equal(m, proto.Clone(m)) {
			t.Fatalf("%T clone lost empty value", m)
		}
		text := proto.CompactTextString(m)
		n = fresh(m)
		if err := proto.UnmarshalText(text, n); err != nil {
			t.Fatal(err)
		}
		if !proto.Equal(m, n) {
			t.Fatalf("%T text round trip lost presence: %s", m, text)
		}
		j, err := (&jsonpb.Marshaler{}).MarshalToString(m)
		if err != nil {
			t.Fatal(err)
		}
		n = fresh(m)
		if err := jsonpb.UnmarshalString(j, n); err != nil {
			t.Fatal(err)
		}
		if !proto.Equal(m, n) {
			t.Fatalf("%T JSON round trip lost presence: %s", m, j)
		}
	}
}

func TestEmptyBytesMerge(t *testing.T) {
	for _, m := range samples() {
		field(m, "Data").SetBytes([]byte{})
		if proto.Equal(m, fresh(m)) {
			t.Fatal("nil and explicit empty bytes compare equal")
		}
		if m.(interface{ Equal(interface{}) bool }).Equal(fresh(m)) {
			t.Fatal("generated Equal lost bytes presence")
		}
		dst := fresh(m)
		field(dst, "Data").SetBytes([]byte("old"))
		proto.Merge(dst, m)
		if field(dst, "Data").IsNil() || field(dst, "Data").Len() != 0 {
			t.Fatal("Merge lost explicit empty bytes")
		}
		b, err := proto.Marshal(dst)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(b, []byte{0x1a, 0}) {
			t.Fatalf("empty bytes wire: %x", b)
		}
	}
}

func TestOptionalUTF8(t *testing.T) {
	for _, m := range samples() {
		field(m, "Label").Set(reflect.ValueOf(proto.String(string([]byte{255}))))
		if _, err := proto.Marshal(m); err == nil {
			t.Fatalf("%T accepted invalid UTF-8", m)
		}
		if err := proto.Unmarshal([]byte{0x12, 1, 255}, fresh(m)); err == nil {
			t.Fatalf("%T decoded invalid UTF-8", m)
		}
	}
}

func TestOriginalSyntheticDescriptors(t *testing.T) {
	for _, m := range samples() {
		b, _ := m.(interface{ Descriptor() ([]byte, []int) }).Descriptor()
		z, err := gzip.NewReader(bytes.NewReader(b))
		if err != nil {
			t.Fatal(err)
		}
		raw, err := ioutil.ReadAll(z)
		z.Close()
		if err != nil {
			t.Fatal(err)
		}
		var file descriptor.FileDescriptorProto
		if err := proto.Unmarshal(raw, &file); err != nil {
			t.Fatal(err)
		}
		sample := file.MessageType[1]
		count := 0
		for _, f := range sample.Field {
			if f.GetProto3Optional() {
				count++
				if f.OneofIndex == nil || !sample.IsSyntheticOneof(*f.OneofIndex) {
					t.Fatal("synthetic metadata lost")
				}
			}
		}
		if count != 17 || len(sample.OneofDecl) != 18 {
			t.Fatalf("descriptor rewritten: %d optional, %d oneofs", count, len(sample.OneofDecl))
		}
	}
}

// Getters must return schema defaults without dereferencing an absent optional.
func TestAbsentGetters(t *testing.T) {
	for _, m := range samples() {
		for _, receiver := range []reflect.Value{reflect.ValueOf(m), reflect.Zero(reflect.TypeOf(m))} {
			for i := 0; i < receiver.NumMethod(); i++ {
				method := receiver.Type().Method(i)
				if strings.HasPrefix(method.Name, "Get") && method.Type.NumIn() == 1 {
					receiver.Method(i).Call(nil)
				}
			}
		}
	}
}
