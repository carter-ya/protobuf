package jsonpb

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"strconv"
	"unicode/utf8"

	"github.com/gogo/protobuf/proto"
	"github.com/gogo/protobuf/types"
)

// MarshalStandard is the policy bridge used by jsonpb/standard. Prefer that
// package's Marshaler; legacy Marshaler fields and behavior are unchanged.
func MarshalStandard(options *Marshaler, out io.Writer, message proto.Message) error {
	if options == nil {
		options = new(Marshaler)
	}
	var buffer bytes.Buffer
	if err := (&jsonEncoder{Marshaler: options, standard: true}).marshal(&buffer, message); err != nil {
		return err
	}
	if err := validateStandardJSON(buffer.Bytes()); err != nil {
		return err
	}
	_, err := out.Write(buffer.Bytes())
	return err
}

// UnmarshalStandard is the policy bridge used by jsonpb/standard. It consumes
// one complete JSON document and resets the destination before parsing it.
func UnmarshalStandard(options *Unmarshaler, in io.Reader, message proto.Message) error {
	if options == nil {
		options = new(Unmarshaler)
	}
	if message == nil || reflect.ValueOf(message).Kind() != reflect.Ptr || reflect.ValueOf(message).IsNil() {
		return fmt.Errorf("Unmarshal called with nil")
	}
	message.Reset()
	dec := json.NewDecoder(in)
	var raw json.RawMessage
	if err := dec.Decode(&raw); err != nil {
		return err
	}
	var extra json.RawMessage
	if err := dec.Decode(&extra); err != io.EOF {
		if err != nil {
			return err
		}
		return fmt.Errorf("multiple JSON values")
	}
	if err := validateStandardJSON(raw); err != nil {
		return err
	}
	if err := (&jsonDecoder{Unmarshaler: options, standard: true}).unmarshalValue(reflect.ValueOf(message).Elem(), raw, nil); err != nil {
		return err
	}
	return checkRequiredFields(message)
}

func acceptsJSONNull(t reflect.Type) bool {
	if t == reflect.TypeOf((*types.Value)(nil)) {
		return true
	}
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t == reflect.TypeOf(types.NullValue(0)) || t == reflect.TypeOf(types.Value{})
}

func validateStandardJSON(raw []byte) error {
	if !utf8.Valid(raw) {
		return fmt.Errorf("invalid UTF-8 JSON")
	}
	// encoding/json replaces lone UTF-16 surrogates. Reject them before decoding.
	for i := 0; i < len(raw); i++ {
		if raw[i] != '"' {
			continue
		}
		for i++; i < len(raw) && raw[i] != '"'; i++ {
			if raw[i] != '\\' {
				continue
			}
			i++
			if i >= len(raw) {
				return fmt.Errorf("unterminated JSON escape")
			}
			if raw[i] != 'u' {
				continue
			}
			if i+4 >= len(raw) {
				return fmt.Errorf("invalid Unicode escape")
			}
			n, err := strconv.ParseUint(string(raw[i+1:i+5]), 16, 16)
			if err != nil {
				return err
			}
			i += 4
			if n >= 0xDC00 && n <= 0xDFFF {
				return fmt.Errorf("unpaired Unicode surrogate")
			}
			if n < 0xD800 || n > 0xDBFF {
				continue
			}
			if i+6 >= len(raw) || raw[i+1] != '\\' || raw[i+2] != 'u' {
				return fmt.Errorf("unpaired Unicode surrogate")
			}
			low, err := strconv.ParseUint(string(raw[i+3:i+7]), 16, 16)
			if err != nil || low < 0xDC00 || low > 0xDFFF {
				return fmt.Errorf("unpaired Unicode surrogate")
			}
			i += 6
		}
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var visit func(int) error
	visit = func(depth int) error {

		token, err := decoder.Token()
		if err != nil {
			return err
		}
		delimiter, ok := token.(json.Delim)
		if !ok {
			return nil
		}
		// Count object/array containers, not their scalar leaves. This bound
		// applies only to the explicit standard entrypoint.
		if depth >= 100 {
			return fmt.Errorf("JSON nesting limit exceeded")
		}
		switch delimiter {
		case '{':
			keys := make(map[string]bool)
			for decoder.More() {
				key, err := decoder.Token()
				if err != nil {
					return err
				}
				name, ok := key.(string)
				if !ok {
					return fmt.Errorf("expected object key")
				}
				if keys[name] {
					return fmt.Errorf("duplicate JSON key %q", name)
				}
				keys[name] = true
				if err := visit(depth + 1); err != nil {
					return err
				}
			}
		case '[':
			for decoder.More() {
				if err := visit(depth + 1); err != nil {
					return err
				}
			}
		default:
			return fmt.Errorf("unexpected JSON delimiter")
		}
		_, err = decoder.Token()
		return err
	}
	if err := visit(0); err != nil {
		return err
	}
	if _, err := decoder.Token(); err != io.EOF {
		if err != nil {
			return err
		}
		return fmt.Errorf("extra JSON token")
	}
	return nil
}
