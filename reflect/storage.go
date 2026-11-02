package gogoreflect

import (
	"fmt"
	"reflect"
)

// fieldAccessor is implemented by generated messages with private storage. It
// returns a pointer to the actual Go field, never a copy or a detached view.
// Keeping the hook free of V2 types also permits the legacy pure-Go runtime to
// use the same generated storage contract.
type fieldAccessor interface {
	XXX_GogoReflectField(int) interface{}
}

func nativeField(v reflect.Value, index int) (reflect.Value, error) {
	raw := v.Elem().Field(index)
	if raw.CanInterface() && raw.CanSet() {
		return raw, nil
	}
	a, ok := v.Interface().(fieldAccessor)
	if !ok {
		return reflect.Value{}, fmt.Errorf("gogoreflect: %v has unexported storage without a generated field accessor", v.Type())
	}
	p := reflect.ValueOf(a.XXX_GogoReflectField(index))
	if !p.IsValid() || p.Kind() != reflect.Ptr || p.IsNil() || p.Type().Elem() != raw.Type() {
		return reflect.Value{}, fmt.Errorf("gogoreflect: invalid generated field accessor for %v field %d", v.Type(), index)
	}
	if p.Pointer() != raw.Addr().Pointer() {
		return reflect.Value{}, fmt.Errorf("gogoreflect: generated field accessor detached from %v field %d", v.Type(), index)
	}
	return p.Elem(), nil
}

func (m *message) nativeField(index int) reflect.Value {
	v, err := nativeField(m.value, index)
	if err != nil {
		panic(err)
	}
	return v
}
