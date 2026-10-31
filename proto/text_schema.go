package proto

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"reflect"
	"strings"
	"sync"
)

// These partial descriptors avoid a dependency cycle on the generated
// descriptor package. Only schema facts absent from Go struct tags are read.
type textFileDescriptor struct {
	Messages []*textMessageDescriptor `protobuf:"bytes,4,rep,name=message_type"`
}
type textMessageDescriptor struct {
	Nested   []*textMessageDescriptor `protobuf:"bytes,3,rep,name=nested_type"`
	Reserved []string                 `protobuf:"bytes,10,rep,name=reserved_name"`
}

func (m *textFileDescriptor) Reset()          { *m = textFileDescriptor{} }
func (*textFileDescriptor) String() string    { return "textFileDescriptor" }
func (*textFileDescriptor) ProtoMessage()     {}
func (m *textMessageDescriptor) Reset()       { *m = textMessageDescriptor{} }
func (*textMessageDescriptor) String() string { return "textMessageDescriptor" }
func (*textMessageDescriptor) ProtoMessage()  {}

type textSchema struct {
	reserved map[string]bool
	err      error
}

var textSchemas sync.Map // reflect.Type -> immutable *textSchema

func standardTextSchema(typ reflect.Type) *textSchema {
	if cached, ok := textSchemas.Load(typ); ok {
		return cached.(*textSchema)
	}
	schema := new(textSchema)
	if source, ok := reflect.New(typ).Interface().(interface{ Descriptor() ([]byte, []int) }); ok {
		schema.reserved, schema.err = readTextReservedNames(source)
	}
	actual, _ := textSchemas.LoadOrStore(typ, schema)
	return actual.(*textSchema)
}

func readTextReservedNames(source interface{ Descriptor() ([]byte, []int) }) (map[string]bool, error) {
	wire, path := source.Descriptor()
	reader, err := gzip.NewReader(bytes.NewReader(wire))
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	// Generated descriptors are trusted schema data, but bound decompression so
	// a malformed hand-written Descriptor method cannot grow without limit.
	const maxDescriptor = 64 << 20
	data, err := ioutil.ReadAll(io.LimitReader(reader, maxDescriptor+1))
	if err != nil {
		return nil, err
	}
	if len(data) > maxDescriptor {
		return nil, fmt.Errorf("text descriptor too large")
	}
	var file textFileDescriptor
	if err := Unmarshal(data, &file); err != nil {
		return nil, err
	}
	messages := file.Messages
	var selected *textMessageDescriptor
	for _, index := range path {
		if index < 0 || index >= len(messages) {
			return nil, fmt.Errorf("invalid text descriptor path")
		}
		selected = messages[index]
		messages = selected.Nested
	}
	if selected == nil {
		return nil, fmt.Errorf("empty text descriptor path")
	}
	names := make(map[string]bool, len(selected.Reserved))
	for _, name := range selected.Reserved {
		names[name] = true
	}
	return names, nil
}

func textIdentifier(value string) bool {
	for index, ch := range value {
		if ch >= 'a' && ch <= 'z' || ch >= 'A' && ch <= 'Z' || ch == '_' || index > 0 && ch >= '0' && ch <= '9' {
			continue
		}
		return false
	}
	return value != ""
}

// The schema does not give reserved names a type. Validate their lexical form
// and balanced structure, as the reference parser's SkipField does.
func (p *textParser) skipReservedField(depth int) error {
	tok := p.next()
	if tok.err != nil {
		return tok.err
	}
	if tok.value == ":" {
		if err := p.skipReservedValue(depth); err != nil {
			return err
		}
	} else if tok.value == "[" {
		// Colons are optional for repeated message values as well as a single
		// message. Without a colon, scalar or nested-list elements are invalid.
		if err := p.skipReservedList(depth+1, true); err != nil {
			return err
		}
	} else if tok.value == "{" || tok.value == "<" {
		p.back()
		if err := p.skipReservedValue(depth); err != nil {
			return err
		}
	} else {
		return p.errorf("expected ':' or message body for reserved field")
	}
	return p.consumeOptionalSeparator()
}

func (p *textParser) skipReservedValue(depth int) error {
	if depth > 100 {
		return p.errorf("reserved text value nesting exceeds 100")
	}
	tok := p.next()
	if tok.err != nil {
		return tok.err
	}
	value := tok.value
	switch value {
	case "{", "<":
		end := "}"
		if value == "<" {
			end = ">"
		}
		for {
			tok = p.next()
			if tok.err != nil {
				return tok.err
			}
			if tok.value == end {
				return nil
			}
			if tok.value == "[" {
				if _, err := p.consumeExtName(); err != nil {
					return err
				}
			} else if !textIdentifier(tok.value) {
				return p.errorf("invalid reserved message field name %q", tok.value)
			}
			if err := p.skipReservedField(depth + 1); err != nil {
				return err
			}
		}
	case "[":
		return p.skipReservedList(depth+1, false)
	}
	if len(value) > 0 && isQuote(value[0]) {
		return nil
	}
	negative := strings.HasPrefix(value, "-")
	if value == "-" {
		tok = p.next()
		if tok.err != nil {
			return tok.err
		}
		value = tok.value
	} else if negative {
		value = value[1:]
	}
	if textIdentifier(value) {
		if !negative {
			return nil
		}
		switch strings.ToLower(value) {
		case "inf", "infinity", "nan":
			return nil
		}
	} else if reservedTextNumber(value) {
		return nil
	}
	return p.errorf("invalid reserved scalar value %q", value)
}

func (p *textParser) skipReservedList(depth int, messagesOnly bool) error {
	if depth > 100 {
		return p.errorf("reserved text value nesting exceeds 100")
	}
	tok := p.next()
	if tok.err != nil {
		return tok.err
	}
	if tok.value == "]" {
		return nil
	}
	for {
		if messagesOnly && tok.value != "{" && tok.value != "<" {
			return p.errorf("expected message body in colonless reserved list")
		}
		p.back()
		if err := p.skipReservedValue(depth); err != nil {
			return err
		}
		tok = p.next()
		if tok.err != nil {
			return tok.err
		}
		if tok.value == "]" {
			return nil
		}
		if tok.value != "," {
			return p.errorf("expected ',' or ']' in reserved list")
		}
		tok = p.next()
		if tok.err != nil {
			return tok.err
		}
	}
}

func reservedTextNumber(value string) bool {
	if value == "" {
		return false
	}
	if strings.HasPrefix(value, "0x") || strings.HasPrefix(value, "0X") {
		if len(value) == 2 {
			return false
		}
		for _, digit := range value[2:] {
			if digit >= '0' && digit <= '9' || digit >= 'a' && digit <= 'f' || digit >= 'A' && digit <= 'F' {
				continue
			}
			return false
		}
		return true
	}
	if len(value) > 1 && value[0] == '0' && value[1] >= '0' && value[1] <= '9' {
		for _, digit := range value {
			if digit < '0' || digit > '7' {
				return false
			}
		}
		return true
	}
	if value[len(value)-1] == 'f' || value[len(value)-1] == 'F' {
		value = value[:len(value)-1]
	}
	return standardDecimalFloat(value)
}
