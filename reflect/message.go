package gogoreflect

import (
	"fmt"
	"math"
	"reflect"

	gogo "github.com/gogo/protobuf/proto"
	"google.golang.org/protobuf/encoding/protowire"
	v2 "google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/runtime/protoiface"
)

// Wrap validates a native representation and exposes a live view. The returned
// message shares storage with m; callers synchronize writes exactly as for m.
// Wrap accepts a typed nil pointer and returns an invalid, read-only message.
func Wrap(m gogo.Message) (protoreflect.ProtoMessage, error) {
	if m == nil {
		return nil, fmt.Errorf("gogoreflect: nil interface")
	}
	info, err := model(reflect.TypeOf(m))
	if err != nil {
		return nil, err
	}
	return &message{info: info, value: reflect.ValueOf(m)}, nil
}

// MessageOf is the generated ProtoReflect entrypoint. Unsupported manually
// supplied representations panic; use Wrap when a fallible API is needed.
func MessageOf(m gogo.Message) protoreflect.Message {
	p, err := Wrap(m)
	if err != nil {
		panic(err)
	}
	return p.ProtoReflect()
}

// Unwrap returns the original gogo message behind this module's live view.
func Unwrap(m protoreflect.Message) (gogo.Message, bool) {
	x, ok := m.(*message)
	if !ok {
		return nil, false
	}
	return x.native(), true
}

type message struct {
	info  *messageType
	value reflect.Value
}

func (m *message) native() gogo.Message                       { return m.value.Interface().(gogo.Message) }
func (m *message) ProtoReflect() protoreflect.Message         { return m }
func (m *message) Descriptor() protoreflect.MessageDescriptor { return m.info.desc }
func (m *message) Type() protoreflect.MessageType             { return m.info }
func (m *message) New() protoreflect.Message                  { return m.info.New() }
func (m *message) Interface() protoreflect.ProtoMessage {
	if p, ok := m.native().(protoreflect.ProtoMessage); ok {
		return p
	}
	return m
}
func (m *message) IsValid() bool { return !m.value.IsNil() }
func (m *message) Reset()        { m.requireMutable(); m.value.Elem().SetZero() }
func (m *message) requireMutable() {
	if !m.IsValid() {
		panic("gogoreflect: mutation of nil message")
	}
}
func (t *messageType) Descriptor() protoreflect.MessageDescriptor { return t.desc }
func (t *messageType) New() protoreflect.Message {
	return &message{info: t, value: reflect.New(t.typ.Elem())}
}
func (t *messageType) Zero() protoreflect.Message {
	return &message{info: t, value: reflect.Zero(t.typ)}
}
func (m *message) field(fd protoreflect.FieldDescriptor) *fieldInfo {
	if fd == nil || fd.IsExtension() || m.info.desc.Fields().ByNumber(fd.Number()) != fd {
		panic("gogoreflect: foreign field descriptor")
	}
	return m.info.fields[fd.Number()]
}
func (m *message) slot(f *fieldInfo) slot {
	return slot{typ: f.typ, get: func() reflect.Value {
		if !m.IsValid() {
			return reflect.Zero(f.typ)
		}
		v := m.value.Elem().Field(f.index)
		if f.wrapper != nil {
			if v.IsNil() || v.Elem().Type() != f.wrapper || v.Elem().IsNil() {
				return reflect.Zero(f.typ)
			}
			return v.Elem().Elem().Field(0)
		}
		return v
	}, set: func(v reflect.Value) {
		m.requireMutable()
		target := m.value.Elem().Field(f.index)
		if f.wrapper != nil {
			w := reflect.New(f.wrapper.Elem())
			w.Elem().Field(0).Set(v)
			target.Set(w)
		} else {
			target.Set(v)
		}
	}}
}
func (m *message) Has(fd protoreflect.FieldDescriptor) bool {
	f := m.field(fd)
	if !m.IsValid() {
		return false
	}
	if f.wrapper != nil {
		v := m.value.Elem().Field(f.index)
		return !v.IsNil() && v.Elem().Type() == f.wrapper && !v.Elem().IsNil()
	}
	v := m.slot(f).get()
	if fd.IsList() || fd.IsMap() {
		return v.Len() > 0
	}
	if fd.HasPresence() {
		return !v.IsNil()
	}
	switch fd.Kind() {
	case protoreflect.BytesKind:
		return v.Len() > 0
	case protoreflect.FloatKind:
		return math.Float32bits(float32(v.Float())) != 0
	case protoreflect.DoubleKind:
		return math.Float64bits(v.Float()) != 0
	}
	return !v.IsZero()
}
func (m *message) Get(fd protoreflect.FieldDescriptor) protoreflect.Value {
	f := m.field(fd)
	s := m.slot(f)
	if fd.IsList() {
		if s.get().Len() == 0 {
			return protoreflect.ValueOfList(&list{fd: fd, slot: detached(f.typ, reflect.Zero(f.typ)), readonly: true})
		}
		return protoreflect.ValueOfList(&list{fd: fd, slot: s})
	}
	if fd.IsMap() {
		if s.get().Len() == 0 {
			return protoreflect.ValueOfMap(&mapping{fd: fd, slot: detached(f.typ, reflect.Zero(f.typ)), readonly: true})
		}
		return protoreflect.ValueOfMap(&mapping{fd: fd, slot: s})
	}
	if !m.Has(fd) {
		if fd.Message() != nil {
			info, err := model(f.typ)
			if err != nil {
				panic(err)
			}
			return protoreflect.ValueOfMessage(info.Zero())
		}
		if fd.Kind() == protoreflect.BytesKind {
			return protoreflect.ValueOfBytes(append([]byte(nil), fd.Default().Bytes()...))
		}
		return fd.Default()
	}
	v := s.get()
	if v.Kind() == reflect.Ptr && fd.Message() == nil {
		v = v.Elem()
	}
	return reflectValue(fd, v)
}
func (m *message) Set(fd protoreflect.FieldDescriptor, v protoreflect.Value) {
	f := m.field(fd)
	s := m.slot(f)
	if fd.Kind() == protoreflect.BytesKind && fd.HasPresence() && (fd.ContainingOneof() == nil || fd.ContainingOneof().IsSynthetic()) && v.Bytes() == nil {
		v = protoreflect.ValueOfBytes([]byte{})
	}
	s.set(convertField(fd, v, f.typ))
}
func (m *message) Clear(fd protoreflect.FieldDescriptor) {
	f := m.field(fd)
	m.requireMutable()
	if f.wrapper != nil {
		if m.Has(fd) {
			m.value.Elem().Field(f.index).SetZero()
		}
		return
	}
	m.slot(f).set(reflect.Zero(f.typ))
}
func (m *message) Range(fn func(protoreflect.FieldDescriptor, protoreflect.Value) bool) {
	for i := 0; i < m.info.desc.Fields().Len(); i++ {
		fd := m.info.desc.Fields().Get(i)
		if m.Has(fd) && !fn(fd, m.Get(fd)) {
			return
		}
	}
}
func (m *message) Mutable(fd protoreflect.FieldDescriptor) protoreflect.Value {
	f := m.field(fd)
	m.requireMutable()
	s := m.slot(f)
	if fd.IsMap() {
		if s.get().IsNil() {
			s.set(reflect.MakeMap(f.typ))
		}
		return protoreflect.ValueOfMap(&mapping{fd: fd, slot: s})
	}
	if fd.IsList() {
		return protoreflect.ValueOfList(&list{fd: fd, slot: s})
	}
	if fd.Message() != nil {
		if !m.Has(fd) {
			s.set(reflect.New(f.typ.Elem()))
		}
		return m.Get(fd)
	}
	panic("gogoreflect: Mutable requires a composite field")
}
func (m *message) NewField(fd protoreflect.FieldDescriptor) protoreflect.Value {
	f := m.field(fd)
	return newValue(fd, f.typ)
}
func (m *message) WhichOneof(od protoreflect.OneofDescriptor) protoreflect.FieldDescriptor {
	if od == nil || m.info.desc.Oneofs().ByName(od.Name()) != od {
		panic("gogoreflect: foreign oneof descriptor")
	}
	for i := 0; i < od.Fields().Len(); i++ {
		fd := od.Fields().Get(i)
		if m.Has(fd) {
			return fd
		}
	}
	return nil
}
func (m *message) GetUnknown() protoreflect.RawFields {
	if !m.IsValid() {
		return nil
	}
	return m.value.Elem().Field(m.info.unknown).Bytes()
}
func (m *message) SetUnknown(b protoreflect.RawFields) {
	m.requireMutable()
	for rest := []byte(b); len(rest) > 0; {
		_, _, n := protowire.ConsumeField(rest)
		if n < 0 {
			panic("gogoreflect: malformed unknown fields")
		}
		rest = rest[n:]
	}
	m.value.Elem().Field(m.info.unknown).SetBytes(b)
}
func (m *message) ProtoMethods() *protoiface.Methods { return nil }

type slot struct {
	typ reflect.Type
	get func() reflect.Value
	set func(reflect.Value)
}

func detached(t reflect.Type, v reflect.Value) slot {
	p := reflect.New(t).Elem()
	p.Set(v)
	return slot{typ: t, get: func() reflect.Value { return p }, set: func(v reflect.Value) { p.Set(v) }}
}
func newValue(fd protoreflect.FieldDescriptor, t reflect.Type) protoreflect.Value {
	if fd.IsList() {
		return protoreflect.ValueOfList(&list{fd: fd, slot: detached(t, reflect.MakeSlice(t, 0, 0))})
	}
	if fd.IsMap() {
		return protoreflect.ValueOfMap(&mapping{fd: fd, slot: detached(t, reflect.MakeMap(t))})
	}
	if fd.Message() != nil {
		info, err := model(t)
		if err != nil {
			panic(err)
		}
		return protoreflect.ValueOfMessage(info.New())
	}
	if fd.Kind() == protoreflect.BytesKind {
		return protoreflect.ValueOfBytes(append([]byte(nil), fd.Default().Bytes()...))
	}
	return fd.Default()
}
func reflectValue(fd protoreflect.FieldDescriptor, v reflect.Value) protoreflect.Value {
	switch fd.Kind() {
	case protoreflect.BoolKind:
		return protoreflect.ValueOfBool(v.Bool())
	case protoreflect.EnumKind:
		return protoreflect.ValueOfEnum(protoreflect.EnumNumber(v.Int()))
	case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind:
		return protoreflect.ValueOfInt32(int32(v.Int()))
	case protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind:
		return protoreflect.ValueOfInt64(v.Int())
	case protoreflect.Uint32Kind, protoreflect.Fixed32Kind:
		return protoreflect.ValueOfUint32(uint32(v.Uint()))
	case protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
		return protoreflect.ValueOfUint64(v.Uint())
	case protoreflect.FloatKind:
		return protoreflect.ValueOfFloat32(float32(v.Float()))
	case protoreflect.DoubleKind:
		return protoreflect.ValueOfFloat64(v.Float())
	case protoreflect.StringKind:
		return protoreflect.ValueOfString(v.String())
	case protoreflect.BytesKind:
		return protoreflect.ValueOfBytes(v.Bytes())
	case protoreflect.MessageKind, protoreflect.GroupKind:
		info, err := model(v.Type())
		if err != nil {
			panic(err)
		}
		return protoreflect.ValueOfMessage(&message{info: info, value: v})
	default:
		panic("gogoreflect: unsupported kind")
	}
}
func convertScalar(fd protoreflect.FieldDescriptor, v protoreflect.Value, t reflect.Type) reflect.Value {
	valid := false
	switch fd.Kind() {
	case protoreflect.BoolKind:
		_, valid = v.Interface().(bool)
	case protoreflect.EnumKind:
		_, valid = v.Interface().(protoreflect.EnumNumber)
	case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind:
		_, valid = v.Interface().(int32)
	case protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind:
		_, valid = v.Interface().(int64)
	case protoreflect.Uint32Kind, protoreflect.Fixed32Kind:
		_, valid = v.Interface().(uint32)
	case protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
		_, valid = v.Interface().(uint64)
	case protoreflect.FloatKind:
		_, valid = v.Interface().(float32)
	case protoreflect.DoubleKind:
		_, valid = v.Interface().(float64)
	case protoreflect.StringKind:
		_, valid = v.Interface().(string)
	case protoreflect.BytesKind:
		_, valid = v.Interface().([]byte)
	case protoreflect.MessageKind, protoreflect.GroupKind:
		_, valid = v.Interface().(protoreflect.Message)
	}
	if !valid {
		panic("gogoreflect: invalid value type for " + string(fd.FullName()))
	}

	out := reflect.New(t).Elem()
	switch fd.Kind() {
	case protoreflect.BoolKind:
		out.SetBool(v.Bool())
	case protoreflect.EnumKind:
		out.SetInt(int64(v.Enum()))
	case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind, protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind:
		out.SetInt(v.Int())
	case protoreflect.Uint32Kind, protoreflect.Fixed32Kind, protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
		out.SetUint(v.Uint())
	case protoreflect.FloatKind, protoreflect.DoubleKind:
		out.SetFloat(v.Float())
	case protoreflect.StringKind:
		out.SetString(v.String())
	case protoreflect.BytesKind:
		out.SetBytes(v.Bytes())
	case protoreflect.MessageKind, protoreflect.GroupKind:
		mv := v.Message()
		if mv.Descriptor() != fd.Message() {
			panic("gogoreflect: message descriptor mismatch")
		}
		if !mv.IsValid() {
			panic("gogoreflect: nil message value")
		}
		if native, ok := Unwrap(mv); ok && reflect.TypeOf(native) == t {
			return reflect.ValueOf(native)
		}
		info, err := model(t)
		if err != nil {
			panic(err)
		}
		fresh := info.New()
		v2.Merge(fresh.Interface(), mv.Interface())
		return fresh.(*message).value
	default:
		panic("gogoreflect: unsupported field kind")
	}
	return out
}
func convertField(fd protoreflect.FieldDescriptor, v protoreflect.Value, t reflect.Type) reflect.Value {
	if fd.IsList() {
		x, ok := v.List().(*list)
		if !ok || !sameValueType(x.fd, fd) || x.slot.typ != t || !x.IsValid() {
			panic("gogoreflect: incompatible list")
		}
		return x.slot.get()
	}
	if fd.IsMap() {
		x, ok := v.Map().(*mapping)
		if !ok || !sameValueType(x.fd, fd) || x.slot.typ != t || !x.IsValid() {
			panic("gogoreflect: incompatible map")
		}
		return x.slot.get()
	}
	if t.Kind() == reflect.Ptr && fd.Message() == nil {
		p := reflect.New(t.Elem())
		p.Elem().Set(convertScalar(fd, v, t.Elem()))
		return p
	}
	return convertScalar(fd, v, t)
}

func sameValueType(a, b protoreflect.FieldDescriptor) bool {
	if a.IsMap() != b.IsMap() || a.IsList() != b.IsList() || a.Kind() != b.Kind() {
		return false
	}
	if a.IsMap() {
		return sameValueType(a.MapKey(), b.MapKey()) && sameValueType(a.MapValue(), b.MapValue())
	}
	return a.Message() == b.Message() && a.Enum() == b.Enum()
}
