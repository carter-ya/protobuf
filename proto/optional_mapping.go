package proto

import (
	"fmt"
	"io"
	"reflect"
	"time"
)

// UnmarshalOptionalStd decodes one proto3 optional WKT occurrence into its
// mapped Go pointer. Generated code passes the address of a nullable field.
// Known WKT fields merge across wire occurrences; inner unknown fields cannot
// be represented by these Go mappings and are discarded.
func UnmarshalOptionalStd(destination interface{}, data []byte) error {
	switch target := destination.(type) {
	case **time.Time:
		if target == nil {
			return fmt.Errorf("proto: nil optional standard target")
		}
		value := new(timestamp)
		if *target != nil {
			var err error
			value, err = timestampProto(**target)
			if err != nil {
				return err
			}
		}
		if err := UnmarshalMerge(data, value); err != nil {
			return err
		}
		result, err := timestampFromProto(value)
		if err != nil {
			return err
		}
		*target = &result
		return nil
	case **time.Duration:
		if target == nil {
			return fmt.Errorf("proto: nil optional standard target")
		}
		value := new(duration)
		if *target != nil {
			value = durationProto(**target)
		}
		if err := UnmarshalMerge(data, value); err != nil {
			return err
		}
		result, err := durationFromProto(value)
		if err != nil {
			return err
		}
		*target = &result
		return nil
	}
	field := reflect.ValueOf(destination)
	if field.Kind() != reflect.Ptr || field.IsNil() || field.Elem().Kind() != reflect.Ptr {
		return fmt.Errorf("proto: invalid optional standard target %T", destination)
	}
	field = field.Elem()
	var message Message
	switch field.Type().Elem().Kind() {
	case reflect.Float64:
		message = new(float64Value)
	case reflect.Float32:
		message = new(float32Value)
	case reflect.Int64:
		message = new(int64Value)
	case reflect.Uint64:
		message = new(uint64Value)
	case reflect.Int32:
		message = new(int32Value)
	case reflect.Uint32:
		message = new(uint32Value)
	case reflect.Bool:
		message = new(boolValue)
	case reflect.String:
		message = new(stringValue)
	case reflect.Slice:
		if field.Type().Elem() != reflect.TypeOf([]byte(nil)) {
			return fmt.Errorf("proto: invalid optional standard target %T", destination)
		}
		message = new(bytesValue)
	default:
		return fmt.Errorf("proto: invalid optional standard target %T", destination)
	}
	value := reflect.ValueOf(message).Elem().Field(0)
	if value.Type() != field.Type().Elem() {
		return fmt.Errorf("proto: invalid optional standard target %T", destination)
	}
	if !field.IsNil() {
		value.Set(field.Elem())
	}
	if err := UnmarshalMerge(data, message); err != nil {
		return err
	}
	result := reflect.New(field.Type().Elem())
	result.Elem().Set(value)
	field.Set(result)
	return nil
}

func makeOptionalStdUnmarshaler(fieldType reflect.Type) unmarshaler {
	return func(data []byte, field pointer, wire int) ([]byte, error) {
		if wire != WireBytes {
			return nil, errInternalBadWireType
		}
		length, n := decodeVarint(data)
		if n == 0 {
			return nil, io.ErrUnexpectedEOF
		}
		data = data[n:]
		if length > uint64(len(data)) {
			return nil, io.ErrUnexpectedEOF
		}
		if err := UnmarshalOptionalStd(field.asPointerTo(fieldType).Interface(), data[:length]); err != nil {
			return nil, err
		}
		return data[length:], nil
	}
}

func optionalStdMessage(value interface{}) (Message, error) {
	switch v := value.(type) {
	case time.Time:
		return timestampProto(v)
	case time.Duration:
		return durationProto(v), nil
	case float64:
		return &float64Value{Value: v}, nil
	case float32:
		return &float32Value{Value: v}, nil
	case int64:
		return &int64Value{Value: v}, nil
	case uint64:
		return &uint64Value{Value: v}, nil
	case int32:
		return &int32Value{Value: v}, nil
	case uint32:
		return &uint32Value{Value: v}, nil
	case bool:
		return &boolValue{Value: v}, nil
	case string:
		return &stringValue{Value: v}, nil
	case []byte:
		return &bytesValue{Value: v}, nil
	default:
		return nil, fmt.Errorf("proto: invalid optional standard value %T", value)
	}
}

// MarshalOptionalStd encodes a present mapped WKT value. It shares the
// reflection codec's UTF-8 and floating-point bit handling with fast bindings.
func MarshalOptionalStd(value interface{}) ([]byte, error) {
	message, err := optionalStdMessage(value)
	if err != nil {
		return nil, err
	}
	return Marshal(message)
}

// SizeOfOptionalStd reports the encoded WKT payload size for generated codecs.
// As with Size, invalid values are reported by Marshal rather than this helper.
func SizeOfOptionalStd(value interface{}) int {
	message, err := optionalStdMessage(value)
	if err != nil {
		return 0
	}
	return Size(message)
}
