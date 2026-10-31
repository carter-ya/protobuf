package proto

import "reflect"

// A map entry can omit its message value. The table codec represents that default
// as nil, so ordinary field decoding never visits the empty message's required
// fields. Validate these defaults without changing the decoded Go representation.
func validateEditionMapMessageDefaults(message Message) error {
	var visit func(reflect.Value, bool) error
	visit = func(value reflect.Value, empty bool) error {
		if !value.IsValid() {
			return nil
		}
		if value.Kind() == reflect.Interface {
			if value.IsNil() {
				return nil
			}
			wrapper := value.Elem()
			if wrapper.Kind() == reflect.Ptr && !wrapper.IsNil() && wrapper.Elem().Kind() == reflect.Struct && wrapper.Elem().NumField() > 0 {
				return visit(wrapper.Elem().Field(0), true)
			}
			return nil
		}
		if value.Kind() == reflect.Ptr {
			if value.Type().Elem().Kind() != reflect.Struct || !value.Type().Implements(protoMessageType) {
				return nil
			}
			if value.IsNil() {
				if empty {
					for _, property := range GetProperties(value.Type().Elem()).Prop {
						if property.Required {
							return &RequiredNotSetError{value.Type().String() + "." + property.OrigName}
						}
					}
				}
				return nil
			}
			value = value.Elem()
		}
		switch value.Kind() {
		case reflect.Struct:
			if !reflect.PtrTo(value.Type()).Implements(protoMessageType) {
				return nil
			}
			properties := GetProperties(value.Type())
			for i, property := range properties.Prop {
				field := value.Type().Field(i)
				if field.Tag.Get("protobuf") == "" && field.Tag.Get("protobuf_oneof") == "" {
					continue
				}
				if property.CustomType != "" || property.CastType != "" || property.StdTime || property.StdDuration || property.WktPointer {
					continue
				}
				if property.Required && isNil(value.Field(i)) {
					return &RequiredNotSetError{value.Type().String() + "." + property.OrigName}
				}
				if err := visit(value.Field(i), false); err != nil {
					return err
				}
			}
		case reflect.Map:
			iterator := value.MapRange()
			for iterator.Next() {
				if err := visit(iterator.Value(), true); err != nil {
					return err
				}
			}
		case reflect.Slice:
			if value.Type().Elem().Kind() == reflect.Uint8 {
				return nil
			}
			for i := 0; i < value.Len(); i++ {
				if err := visit(value.Index(i), true); err != nil {
					return err
				}
			}
		}
		return nil
	}
	return visit(reflect.ValueOf(message), false)
}
