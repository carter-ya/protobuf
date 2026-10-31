package proto

import (
	"reflect"
	"sync"
)

var editionExtensionGraphTypes sync.Map // map[reflect.Type]bool, independent of the live registry

// Cache only structural reachability, not registered descriptors: extensions may
// be registered after a codec is initialized. Cyclic message types are allowed.
func hasEditionExtensionGraph(typ reflect.Type) bool {
	if cached, ok := editionExtensionGraphTypes.Load(typ); ok {
		return cached.(bool)
	}
	seen := make(map[reflect.Type]bool)
	var visit func(reflect.Type) bool
	visit = func(t reflect.Type) bool {
		switch t.Kind() {
		case reflect.Ptr, reflect.Slice:
			return visit(t.Elem())
		case reflect.Map:
			return visit(t.Elem())
		}
		if t.Kind() != reflect.Struct || !reflect.PtrTo(t).Implements(protoMessageType) || seen[t] {
			return false
		}
		seen[t] = true
		if _, ok := reflect.PtrTo(t).MethodByName("ExtensionRangeArray"); ok {
			return true
		}
		properties := GetProperties(t)
		for i, property := range properties.Prop {
			field := t.Field(i)
			if property.CustomType != "" || property.CastType != "" || property.StdTime || property.StdDuration || property.WktPointer {
				continue
			}
			if field.Tag.Get("protobuf") != "" && visit(field.Type) {
				return true
			}
		}
		for _, oneof := range properties.OneofTypes {
			if visit(oneof.Type.Elem().Field(0).Type) {
				return true
			}
		}
		return false
	}
	result := visit(typ)
	editionExtensionGraphTypes.Store(typ, result)
	return result
}
