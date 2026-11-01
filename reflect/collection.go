package gogoreflect

import (
	"google.golang.org/protobuf/reflect/protoreflect"
	"reflect"
)

type list struct {
	fd       protoreflect.FieldDescriptor
	slot     slot
	readonly bool
}

func (l *list) Len() int                     { return l.slot.get().Len() }
func (l *list) Get(i int) protoreflect.Value { return reflectValue(l.fd, l.slot.get().Index(i)) }
func (l *list) Set(i int, v protoreflect.Value) {
	l.requireMutable()
	s := l.slot.get()
	s.Index(i).Set(convertScalar(l.fd, v, l.slot.typ.Elem()))
	l.slot.set(s)
}
func (l *list) Append(v protoreflect.Value) {
	l.requireMutable()
	l.slot.set(reflect.Append(l.slot.get(), convertScalar(l.fd, v, l.slot.typ.Elem())))
}
func (l *list) AppendMutable() protoreflect.Value {
	if l.fd.Message() == nil {
		panic("gogoreflect: AppendMutable requires messages")
	}
	v := l.NewElement()
	l.Append(v)
	return v
}
func (l *list) Truncate(n int) {
	l.requireMutable()
	s := l.slot.get()
	if n < 0 || n > s.Len() {
		panic("gogoreflect: invalid truncate")
	}
	for i := n; i < s.Len(); i++ {
		s.Index(i).SetZero()
	}
	l.slot.set(s.Slice(0, n))
}
func (l *list) NewElement() protoreflect.Value {
	if l.fd.Message() != nil {
		info, err := model(l.slot.typ.Elem())
		if err != nil {
			panic(err)
		}
		return protoreflect.ValueOfMessage(info.New())
	}
	if l.fd.Enum() != nil {
		return protoreflect.ValueOfEnum(l.fd.Enum().Values().Get(0).Number())
	}
	return reflectValue(l.fd, reflect.Zero(l.slot.typ.Elem()))
}
func (l *list) IsValid() bool { return !l.readonly }

func (l *list) requireMutable() {
	if !l.IsValid() {
		panic("gogoreflect: mutation of read-only list")
	}
}

type mapping struct {
	fd       protoreflect.FieldDescriptor
	slot     slot
	readonly bool
}

func (m *mapping) Len() int { return m.slot.get().Len() }
func (m *mapping) key(k protoreflect.MapKey) reflect.Value {
	return convertScalar(m.fd.MapKey(), k.Value(), m.slot.typ.Key())
}
func (m *mapping) Range(fn func(protoreflect.MapKey, protoreflect.Value) bool) {
	v := m.slot.get()
	for _, k := range v.MapKeys() {
		x := v.MapIndex(k)
		if x.IsValid() && !fn(reflectValue(m.fd.MapKey(), k).MapKey(), reflectValue(m.fd.MapValue(), x)) {
			return
		}
	}
}
func (m *mapping) Has(k protoreflect.MapKey) bool { return m.slot.get().MapIndex(m.key(k)).IsValid() }
func (m *mapping) Clear(k protoreflect.MapKey) {
	if !m.IsValid() {
		panic("gogoreflect: mutation of read-only map")
	}
	v := m.slot.get()
	v.SetMapIndex(m.key(k), reflect.Value{})
	m.slot.set(v)
}
func (m *mapping) Get(k protoreflect.MapKey) protoreflect.Value {
	v := m.slot.get().MapIndex(m.key(k))
	if !v.IsValid() {
		return protoreflect.Value{}
	}
	return reflectValue(m.fd.MapValue(), v)
}
func (m *mapping) Set(k protoreflect.MapKey, v protoreflect.Value) {
	x := convertScalar(m.fd.MapValue(), v, m.slot.typ.Elem())
	s := m.slot.get()
	if s.IsNil() {
		panic("gogoreflect: mutation of read-only map")
	}
	s.SetMapIndex(m.key(k), x)
	m.slot.set(s)
}
func (m *mapping) Mutable(k protoreflect.MapKey) protoreflect.Value {
	if m.fd.MapValue().Message() == nil {
		panic("gogoreflect: Mutable requires message values")
	}
	if !m.Has(k) || !m.Get(k).Message().IsValid() {
		m.Set(k, m.NewValue())
	}
	return m.Get(k)
}
func (m *mapping) NewValue() protoreflect.Value { return newValue(m.fd.MapValue(), m.slot.typ.Elem()) }
func (m *mapping) IsValid() bool                { return !m.readonly }
