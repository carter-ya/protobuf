package proto

import (
	"reflect"
	"strings"
)

func optionalCastTags(tags []string) bool {
	optional, cast := false, false
	for _, tag := range tags {
		optional = optional || tag == "proto3_optional"
		cast = cast || strings.HasPrefix(tag, "casttype=")
	}
	return optional && cast
}

// Optional casts use the same Go kind and width as their wire scalar. The
// reflective pointer implementation cannot assert a pointer to a named type
// as a builtin pointer, so codec tables use a temporary builtin field slot.
func optionalCastType(fieldType reflect.Type) reflect.Type {
	base := fieldType
	pointer := base.Kind() == reflect.Ptr
	if pointer {
		base = base.Elem()
	}
	var builtin reflect.Type
	switch base.Kind() {
	case reflect.Bool:
		builtin = reflect.TypeOf(false)
	case reflect.Float32:
		builtin = reflect.TypeOf(float32(0))
	case reflect.Float64:
		builtin = reflect.TypeOf(float64(0))
	case reflect.Int32:
		builtin = reflect.TypeOf(int32(0))
	case reflect.Int64:
		builtin = reflect.TypeOf(int64(0))
	case reflect.Uint32:
		builtin = reflect.TypeOf(uint32(0))
	case reflect.Uint64:
		builtin = reflect.TypeOf(uint64(0))
	case reflect.String:
		builtin = reflect.TypeOf("")
	case reflect.Slice:
		if base.Elem().Kind() == reflect.Uint8 {
			builtin = reflect.TypeOf([]byte(nil))
		}
	}
	if builtin == nil {
		return fieldType
	}
	if pointer {
		return reflect.PtrTo(builtin)
	}
	return builtin
}

func optionalCastPointer(field pointer, fieldType, builtinType reflect.Type) pointer {
	value := field.asPointerTo(fieldType).Elem()
	slot := reflect.New(builtinType)
	slot.Elem().Set(value.Convert(builtinType))
	return valToPointer(slot)
}
