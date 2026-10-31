package proto

import (
	"bytes"
	"encoding"
	"fmt"
	"io"
	"math"
	"reflect"
	"strconv"
	"strings"
)

// DefaultTextRecursionLimit matches the official Go text decoder's default.
const DefaultTextRecursionLimit = 10000

func standardTextLimit(limit int) (int, error) {
	if limit < 0 {
		return 0, fmt.Errorf("proto: negative standard text recursion limit")
	}
	if limit == 0 {
		return DefaultTextRecursionLimit, nil
	}
	return limit, nil
}

// MarshalTextStandard is the policy bridge for textpb/standard. Historical
// TextMarshaler fields and defaults are unchanged. Output is committed only
// after successful encoding; custom TextMarshaler hooks own their semantics.
func MarshalTextStandard(options *TextMarshaler, out io.Writer, message Message, printUnknown bool) error {
	return MarshalTextStandardWithLimit(options, out, message, printUnknown, 0)
}

// MarshalTextStandardWithLimit additionally bounds message nesting. Zero uses
// DefaultTextRecursionLimit. Cyclic Go message graphs are rejected independently.
func MarshalTextStandardWithLimit(options *TextMarshaler, out io.Writer, message Message, printUnknown bool, limit int) error {
	limit, err := standardTextLimit(limit)
	if err != nil {
		return err
	}
	if message == nil || reflect.ValueOf(message).Kind() != reflect.Ptr || reflect.ValueOf(message).IsNil() {
		return fmt.Errorf("proto: nil standard text message")
	}
	if options == nil {
		options = new(TextMarshaler)
	}
	var buffer bytes.Buffer
	if err := options.marshal(&buffer, message, true, printUnknown, limit); err != nil {
		return err
	}
	_, err = out.Write(buffer.Bytes())
	return err
}

// UnmarshalTextStandard parses one message with standard numeric, enum and
// string rules. It resets the destination, like UnmarshalText. A custom
// TextUnmarshaler remains responsible for its own representation and policy.
func UnmarshalTextStandard(input string, message Message) error {
	return UnmarshalTextStandardWithLimit(input, message, 0)
}

// UnmarshalTextStandardWithLimit bounds message nesting without changing the
// historical text parser. Zero uses DefaultTextRecursionLimit.
func UnmarshalTextStandardWithLimit(input string, message Message, limit int) error {
	limit, err := standardTextLimit(limit)
	if err != nil {
		return err
	}
	v := reflect.ValueOf(message)
	if message == nil || v.Kind() != reflect.Ptr || v.IsNil() {
		return fmt.Errorf("proto: nil standard text message")
	}
	if custom, ok := message.(encoding.TextUnmarshaler); ok {
		return custom.UnmarshalText([]byte(input))
	}
	message.Reset()
	parser := newTextParser(input)
	parser.standard = true
	parser.recursionLimit = limit
	return parser.readStruct(v.Elem(), "")
}

// A decimal float follows the text-format grammar, not Go's hexadecimal float
// grammar. Decimal integers are also permitted; leading-zero integers are octal
// and cannot be cast to float. A suffix applies only to decimal literals.
func parseStandardTextFloat(input string, bits int) (float64, error) {
	negative := false
	value := input
	if strings.HasPrefix(value, "-") {
		negative = true
		value = value[1:]
	}
	switch strings.ToLower(value) {
	case "inf", "infinity":
		if negative {
			return math.Inf(-1), nil
		}
		return math.Inf(1), nil
	case "nan":
		return math.NaN(), nil
	}
	if len(value) != 0 && (value[len(value)-1] == 'f' || value[len(value)-1] == 'F') {
		value = value[:len(value)-1]
	}
	if !standardDecimalFloat(value) {
		return 0, fmt.Errorf("invalid decimal float %q", input)
	}
	if negative {
		value = "-" + value
	}
	number, err := strconv.ParseFloat(value, bits)
	if err != nil {
		if rangeError, ok := err.(*strconv.NumError); ok && rangeError.Err == strconv.ErrRange && math.IsInf(number, 0) {
			return number, nil
		}
	}
	return number, err
}

func standardDecimalFloat(value string) bool {
	position := 0
	for position < len(value) && value[position] >= '0' && value[position] <= '9' {
		position++
	}
	integer := position
	if integer > 1 && value[0] == '0' {
		return false
	}
	fraction := 0
	if position < len(value) && value[position] == '.' {
		position++
		start := position
		for position < len(value) && value[position] >= '0' && value[position] <= '9' {
			position++
		}
		fraction = position - start
	}
	if integer == 0 && fraction == 0 {
		return false
	}
	if position < len(value) && (value[position] == 'e' || value[position] == 'E') {
		position++
		if position < len(value) && (value[position] == '+' || value[position] == '-') {
			position++
		}
		start := position
		for position < len(value) && value[position] >= '0' && value[position] <= '9' {
			position++
		}
		if position == start {
			return false
		}
	}
	return position == len(value)
}

func knownTextEnumNumber(properties *Properties, number int32) bool {
	if len(properties.ClosedEnum) != 0 {
		return properties.ClosedEnum[number]
	}
	if properties.proto3 || properties.Enum == "" {
		return true
	}
	for _, declared := range EnumValueMap(properties.Enum) {
		if declared == number {
			return true
		}
	}
	return false
}

// Matches the reference tokenizer's URL character mode, scoped to brackets.
func standardURLChar(ch byte) bool {
	return ch >= 'a' && ch <= 'z' || ch >= 'A' && ch <= 'Z' || ch >= '0' && ch <= '9' || strings.IndexByte("_-.~!$&()*+,;=%/", ch) >= 0
}

func validStandardTextName(value string) bool {
	name := value
	if slash := strings.LastIndexByte(value, '/'); slash >= 0 {
		if slash == 0 || value[0] == '/' {
			return false
		}
		prefix := value[:slash]
		for i := 0; i < len(prefix); i++ {
			if !standardURLChar(prefix[i]) {
				return false
			}
			if prefix[i] == '%' {
				if i+2 >= len(prefix) || !textHexDigit(prefix[i+1]) || !textHexDigit(prefix[i+2]) {
					return false
				}
				i += 2
			}
		}
		name = value[slash+1:]
	}
	for _, identifier := range strings.Split(name, ".") {
		if !textIdentifier(identifier) {
			return false
		}
	}
	return true
}

func textHexDigit(ch byte) bool {
	return ch >= '0' && ch <= '9' || ch >= 'a' && ch <= 'f' || ch >= 'A' && ch <= 'F'
}
