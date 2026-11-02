package proto

import (
	"github.com/gogo/protobuf/internal/storage"
	"reflect"
)

func generatedField(v reflect.Value, index int) reflect.Value { return storage.Field(v, index) }

// Resolve Any's wire fields independently of generated Go storage names.
func anyStorageFields(v reflect.Value) (reflect.Value, reflect.Value, bool) {
	props := GetProperties(v.Type())
	typeIndex, typeOK := props.decoderTags.get(1)
	valueIndex, valueOK := props.decoderTags.get(2)
	if !typeOK || !valueOK {
		return reflect.Value{}, reflect.Value{}, false
	}
	typeURL, value := generatedField(v, typeIndex), generatedField(v, valueIndex)
	valid := typeURL.Kind() == reflect.String && value.Kind() == reflect.Slice && value.Type().Elem().Kind() == reflect.Uint8
	return typeURL, value, valid
}
