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

// MarshalTextStandard is the policy bridge for textpb/standard. Historical
// TextMarshaler fields and defaults are unchanged. Output is committed only
// after successful encoding; custom TextMarshaler hooks own their semantics.
func MarshalTextStandard(options *TextMarshaler, out io.Writer, message Message, printUnknown bool) error {
	if message == nil || reflect.ValueOf(message).Kind() != reflect.Ptr || reflect.ValueOf(message).IsNil() {
		return fmt.Errorf("proto: nil standard text message")
	}
	if options == nil {
		options = new(TextMarshaler)
	}
	var buffer bytes.Buffer
	if err := options.marshal(&buffer, message, true, printUnknown); err != nil {
		return err
	}
	_, err := out.Write(buffer.Bytes())
	return err
}

// UnmarshalTextStandard parses one message with standard numeric, enum and
// string rules. It resets the destination, like UnmarshalText. A custom
// TextUnmarshaler remains responsible for its own representation and policy.
func UnmarshalTextStandard(input string, message Message) error {
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
