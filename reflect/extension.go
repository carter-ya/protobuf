package gogoreflect

import (
	"fmt"
	"reflect"
	"strings"
	"sync"

	gogo "github.com/gogo/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

const _ = gogo.GoGoProtoPackageIsVersion17

var extensions sync.Map // *gogo.ExtensionDesc -> *extensionType

type extensionType struct {
	desc   protoreflect.ExtensionDescriptor
	native *gogo.ExtensionDesc
	typ    reflect.Type
}
type extensionDescriptor struct {
	protoreflect.ExtensionDescriptor
	owner *extensionType
}

func (d extensionDescriptor) Type() protoreflect.ExtensionType { return d.owner }
func (d extensionDescriptor) Descriptor() protoreflect.ExtensionDescriptor {
	return d.ExtensionDescriptor
}
func (x *extensionType) TypeDescriptor() protoreflect.ExtensionTypeDescriptor {
	return extensionDescriptor{x.desc, x}
}
func (x *extensionType) New() protoreflect.Value { return newValue(x.desc, x.typ) }
func (x *extensionType) Zero() protoreflect.Value {
	if x.desc.IsList() {
		return protoreflect.ValueOfList(&list{fd: x.desc, slot: detached(x.typ, reflect.Zero(x.typ)), readonly: true})
	}
	if x.desc.Message() != nil {
		m, e := model(x.typ)
		if e != nil {
			panic(e)
		}
		return protoreflect.ValueOfMessage(m.Zero())
	}
	return x.desc.Default()
}
func (x *extensionType) ValueOf(v interface{}) protoreflect.Value {
	t := x.typ
	if t.Kind() == reflect.Ptr && x.desc.Message() == nil {
		t = t.Elem()
	}
	rv := reflect.ValueOf(v)
	if !rv.IsValid() || rv.Type() != t {
		panic("gogoreflect: invalid extension Go value")
	}
	if x.desc.IsList() {
		return protoreflect.ValueOfList(&list{fd: x.desc, slot: detached(t, rv)})
	}
	return reflectValue(x.desc, rv)
}
func (x *extensionType) InterfaceOf(v protoreflect.Value) interface{} {
	t := x.typ
	if t.Kind() == reflect.Ptr && x.desc.Message() == nil {
		t = t.Elem()
	}
	return convertField(x.desc, v, t).Interface()
}
func (x *extensionType) IsValidValue(v protoreflect.Value) (valid bool) {
	defer func() {
		if recover() != nil {
			valid = false
		}
	}()
	x.InterfaceOf(v)
	return true
}
func (x *extensionType) IsValidInterface(v interface{}) (valid bool) {
	defer func() {
		if recover() != nil {
			valid = false
		}
	}()
	x.ValueOf(v)
	return true
}

func extensionFor(e *gogo.ExtensionDesc) (*extensionType, error) {
	if x, ok := extensions.Load(e); ok {
		return x.(*extensionType), nil
	}
	typ := reflect.TypeOf(e.ExtensionType)
	if typ == nil || e.ExtendedType == nil {
		return nil, fmt.Errorf("gogoreflect: incomplete extension %d", e.Field)
	}
	schemas.Lock()
	raw := source(e.Filename)
	var err error
	if len(raw) == 0 {
		err = fmt.Errorf("gogoreflect: missing extension source %q", e.Filename)
	} else {
		_, err = schemas.load(raw, e.Filename, make(map[string]bool))
	}
	var d protoreflect.Descriptor
	if err == nil {
		d, err = schemas.files.FindDescriptorByName(protoreflect.FullName(strings.TrimPrefix(e.Name, ".")))
	}
	schemas.Unlock()
	if err != nil {
		return nil, err
	}
	fd, ok := d.(protoreflect.ExtensionDescriptor)
	if !ok || int32(fd.Number()) != e.Field {
		return nil, fmt.Errorf("gogoreflect: extension descriptor mismatch %s", e.Name)
	}
	parent, err := model(reflect.TypeOf(e.ExtendedType))
	if err != nil {
		return nil, err
	}
	if fd.ContainingMessage() != parent.desc {
		return nil, fmt.Errorf("gogoreflect: extension owner mismatch")
	}
	var prop gogo.Properties
	prop.Parse(e.Tag)
	if prop.CustomType != "" || prop.StdTime || prop.StdDuration || prop.WktPointer {
		return nil, fmt.Errorf("gogoreflect: custom/std extension %s", fd.FullName())
	}
	elem := typ
	if fd.IsList() {
		if typ.Kind() != reflect.Slice {
			return nil, fmt.Errorf("gogoreflect: invalid repeated extension storage")
		}
		elem = typ.Elem()
	} else if fd.Message() == nil && fd.Kind() != protoreflect.BytesKind {
		if typ.Kind() != reflect.Ptr {
			return nil, fmt.Errorf("gogoreflect: extension presence lost")
		}
		elem = typ.Elem()
	}
	if fd.Message() != nil {
		child, e := model(elem)
		if e != nil {
			return nil, e
		}
		if child.desc != fd.Message() {
			return nil, fmt.Errorf("gogoreflect: extension message descriptor mismatch")
		}
	} else if err = checkScalar(fd, elem); err != nil {
		return nil, err
	}
	if fd.Enum() != nil && fd.Enum().IsClosed() && !strings.Contains(e.Tag, "enum_closed=") {
		return nil, fmt.Errorf("gogoreflect: extension %s requires enforce_closed_enums regeneration", fd.FullName())
	}
	x := &extensionType{fd, e, typ}
	actual, _ := extensions.LoadOrStore(e, x)
	return actual.(*extensionType), nil
}
func (m *message) extension(fd protoreflect.FieldDescriptor) *extensionType {
	if fd == nil || !fd.IsExtension() || fd.ContainingMessage() != m.info.desc {
		panic("gogoreflect: foreign extension descriptor")
	}
	e := gogo.RegisteredExtensions(m.native())[int32(fd.Number())]
	if e == nil {
		panic("gogoreflect: extension has no native binding")
	}
	x, err := extensionFor(e)
	if err != nil {
		panic(err)
	}
	base := fd
	if td, ok := fd.(protoreflect.ExtensionTypeDescriptor); ok {
		base = td.Descriptor()
	}
	if base != x.desc {
		panic("gogoreflect: mismatched extension descriptor")
	}
	return x
}
func (m *message) extensionSlot(x *extensionType) slot {
	return slot{typ: x.typ, get: func() reflect.Value {
		if !m.IsValid() || !gogo.HasExtension(m.native(), x.native) {
			return reflect.Zero(x.typ)
		}
		v, err := gogo.GetExtension(m.native(), x.native)
		if err != nil {
			panic(err)
		}
		return reflect.ValueOf(v)
	}, set: func(v reflect.Value) {
		m.requireMutable()
		if x.desc.IsList() && v.IsNil() {
			v = reflect.MakeSlice(x.typ, 0, 0)
		}
		if err := gogo.SetExtension(m.native(), x.native, v.Interface()); err != nil {
			panic(err)
		}
	}}
}
