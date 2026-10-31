package proto

import (
	"fmt"
	"io"
	"reflect"
)

func editionExtensionPolicy(extension *ExtensionDesc) bool {
	return extension != nil && extensionProperties(extension).editionExtension
}

func nativeEditionMessageExtension(extension *ExtensionDesc) (reflect.Type, bool) {
	if !editionExtensionPolicy(extension) {
		return nil, false
	}
	properties := extensionProperties(extension)
	if properties.CustomType != "" || properties.CastType != "" || properties.StdTime || properties.StdDuration || properties.WktPointer {
		return nil, false
	}
	typ := reflect.TypeOf(extension.ExtensionType)
	if typ == nil {
		return nil, false
	}
	if typ.Kind() == reflect.Slice {
		typ = typ.Elem()
	}
	if typ.Kind() != reflect.Ptr || typ.Elem().Kind() != reflect.Struct || !typ.Implements(protoMessageType) {
		return nil, false
	}
	return typ, true
}

// Decode each repeated element independently. Singular message fragments must
// merge before checking required fields, including fragments from earlier calls.
func decodeEditionMessageExtension(data []byte, extension *ExtensionDesc) (interface{}, error) {
	element, _ := nativeEditionMessageExtension(extension)
	repeated := reflect.TypeOf(extension.ExtensionType).Kind() == reflect.Slice
	var result reflect.Value
	if repeated {
		result = reflect.MakeSlice(reflect.TypeOf(extension.ExtensionType), 0, 0)
	}
	var merged []byte
	var first reflect.Value
	var firstError error
	fragments := 0
	if len(data) == 0 {
		return nil, io.ErrUnexpectedEOF
	}
	properties := extensionProperties(extension)
	expected := WireBytes
	if properties.Wire == "group" {
		expected = WireStartGroup
	}
	for len(data) != 0 {
		end, start, close, err := editionField(data)
		if err != nil {
			return nil, err
		}
		key, _, err := editionVarint(data)
		if err != nil {
			return nil, err
		}
		if key>>3 != uint64(extension.Field) || int(key&7) != expected {
			return nil, fmt.Errorf("proto: wrong wire type for Editions extension %s", extension.Name)
		}
		body := data[start:close]
		if expected == WireBytes {
			_, prefix, err := editionVarint(body)
			if err != nil {
				return nil, err
			}
			body = body[prefix:]
		}
		value := reflect.New(element.Elem())
		decodeError := unmarshalEditionExtensionValue(body, value.Interface().(Message))
		if decodeError != nil {
			if _, required := decodeError.(*RequiredNotSetError); repeated || !required {
				return nil, decodeError
			}
		}
		if repeated {
			result = reflect.Append(result, value)
		} else {
			fragments++
			if fragments == 1 {
				first, firstError = value, decodeError
				// Capping capacity prevents a subsequent append from modifying
				// the original encoded store through this temporary body slice.
				merged = body[:len(body):len(body)]
			} else {
				merged = append(merged, body...)
			}
		}
		data = data[end:]
	}
	if repeated {
		return result.Interface(), nil
	}
	if fragments == 1 {
		return first.Interface(), firstError
	}
	value := reflect.New(element.Elem())
	if err := unmarshalEditionExtensionValue(merged, value.Interface().(Message)); err != nil {
		return nil, err
	}
	return value.Interface(), nil
}

// A decoded cache must be materialized before additional encoded fragments are
// appended. Only new Editions metadata selects this behavior; legacy caches keep
// their historical policy. Invalid decoded values return errors without dropping
// fields from the cache.
func materializeEditionExtension(value Extension, extension *ExtensionDesc) (Extension, error) {
	if value.value == nil || !editionExtensionPolicy(extension) {
		return value, nil
	}
	if value.desc != nil && value.desc != extension {
		return value, fmt.Errorf("proto: descriptor conflict for Editions extension %s", extension.Name)
	}
	encoded, err := encodeExtension(extension, value.value)
	if err != nil {
		return value, err
	}
	value.enc = append(encoded, value.enc...)
	value.value = nil
	value.desc = extension
	return value, nil
}

// AppendEditionExtension preserves a decoded extension when new fragments are
// appended by generated Editions codecs. Its error is part of binary decoding.
func AppendEditionExtension(message Message, number int32, data []byte) error {
	if bytes, ok := message.(extensionsBytes); ok {
		*bytes.GetExtensions() = append(*bytes.GetExtensions(), data...)
		return nil
	}
	base, err := extendable(message)
	if err != nil {
		return err
	}
	values := base.extensionsWrite()
	value, err := materializeEditionExtension(values[number], RegisteredExtensions(message)[number])
	if err != nil {
		return err
	}
	value.enc = append(value.enc, data...)
	values[number] = value
	return nil
}

// ValidateEditionMessageExtensions checks registered native message payloads
// without replacing their lazy store. Wire, UTF-8 and final required-field
// validation match ordinary messages. Unknown and legacy registrations are opaque.
func ValidateEditionMessageExtensions(message Message) error {
	registered := RegisteredExtensions(message)
	needed := false
	for _, extension := range registered {
		if _, ok := nativeEditionMessageExtension(extension); ok {
			needed = true
			break
		}
	}
	if !needed {
		return nil
	}
	var values map[int32]Extension
	if bytes, ok := message.(extensionsBytes); ok {
		var err error
		values, err = editionExtensionsFromBytes(*bytes.GetExtensions())
		if err != nil {
			return err
		}
	} else {
		base, err := extendable(message)
		if err != nil {
			return err
		}
		store, lock := base.extensionsRead()
		if lock != nil {
			lock.Lock()
			defer lock.Unlock()
		}
		values = store
	}
	for number, value := range values {
		extension := registered[number]
		if _, ok := nativeEditionMessageExtension(extension); !ok {
			continue
		}
		var err error
		value, err = materializeEditionExtension(value, extension)
		if err != nil {
			return err
		}
		if _, err := decodeEditionMessageExtension(value.enc, extension); err != nil {
			return err
		}
	}
	return nil
}

// Unlike the historical byte-store adapter, preserve every occurrence of a
// repeated field or singular message extension rather than keeping only the last.
func editionExtensionsFromBytes(data []byte) (map[int32]Extension, error) {
	values := make(map[int32]Extension)
	for len(data) > 0 {
		size, err := EditionFieldSize(data)
		if err != nil {
			return nil, err
		}
		key, _, err := editionVarint(data)
		if err != nil {
			return nil, err
		}
		number := int32(key >> 3)
		value := values[number]
		value.enc = append(value.enc, data[:size]...)
		values[number] = value
		data = data[size:]
	}
	return values, nil
}

// Generated fast decoders may stop at a missing-required error in an early
// submessage fragment. The table decoder consumes every fragment and preserves
// nonfatal required errors; final encoding then checks the complete merged graph.
func unmarshalEditionExtensionValue(data []byte, message Message) error {
	err := Unmarshal(data, message)
	if err == nil {
		return validateEditionMapMessageDefaults(message)
	}
	if _, required := err.(*RequiredNotSetError); !required {
		return err
	}
	message.Reset()
	var table InternalMessageInfo
	err = table.Unmarshal(message, data)
	if err != nil {
		if _, required := err.(*RequiredNotSetError); !required {
			return err
		}
	}
	_, err = Marshal(message)
	if err != nil {
		return err
	}
	return validateEditionMapMessageDefaults(message)
}
