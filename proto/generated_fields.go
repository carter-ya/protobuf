package proto

import "reflect"

// generatedField preserves the normal reflect path for historical layouts.
// New private fields are accessible only through a generated pointer to their
// actual storage; no unsafe conversion or exported shadow message is needed.
func generatedField(v reflect.Value, index int) reflect.Value {
	f := v.Field(index)
	if f.CanInterface() || !v.CanAddr() || !v.Addr().CanInterface() {
		return f
	}
	a, ok := v.Addr().Interface().(interface{ XXX_GogoReflectField(int) interface{} })
	if !ok {
		return f
	}
	p := reflect.ValueOf(a.XXX_GogoReflectField(index))
	if !p.IsValid() || p.Kind() != reflect.Ptr || p.IsNil() || p.Type().Elem() != f.Type() || p.Pointer() != f.Addr().Pointer() {
		panic("proto: invalid generated private field accessor")
	}
	return p.Elem()
}
