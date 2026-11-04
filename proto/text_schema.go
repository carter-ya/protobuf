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
	Package  string                   `protobuf:"bytes,2,opt,name=package"`
	Messages []*textMessageDescriptor `protobuf:"bytes,4,rep,name=message_type"`
}
type textMessageDescriptor struct {
	Name     string                   `protobuf:"bytes,1,opt,name=name"`
	Fields   []*textFieldDescriptor   `protobuf:"bytes,2,rep,name=field"`
	Nested   []*textMessageDescriptor `protobuf:"bytes,3,rep,name=nested_type"`
	Reserved []string                 `protobuf:"bytes,10,rep,name=reserved_name"`
}

type textFieldDescriptor struct {
	Name     string `protobuf:"bytes,1,opt,name=name"`
	Number   int32  `protobuf:"varint,3,opt,name=number"`
	Type     int32  `protobuf:"varint,5,opt,name=type"`
	TypeName string `protobuf:"bytes,6,opt,name=type_name"`
}

func (m *textFieldDescriptor) Reset()       { *m = textFieldDescriptor{} }
func (*textFieldDescriptor) String() string { return "textFieldDescriptor" }
func (*textFieldDescriptor) ProtoMessage()  {}

func (m *textFileDescriptor) Reset()          { *m = textFileDescriptor{} }
func (*textFileDescriptor) String() string    { return "textFileDescriptor" }
func (*textFileDescriptor) ProtoMessage()     {}
func (m *textMessageDescriptor) Reset()       { *m = textMessageDescriptor{} }
func (*textMessageDescriptor) String() string { return "textMessageDescriptor" }
func (*textMessageDescriptor) ProtoMessage()  {}

type textSchema struct {
	reserved   map[string]bool
	tagsByName map[string]int
	groupNames map[int]string
	err        error
}

var textSchemas sync.Map // reflect.Type -> immutable *textSchema

func standardTextSchema(typ reflect.Type) *textSchema {
	if cached, ok := textSchemas.Load(typ); ok {
		return cached.(*textSchema)
	}
	schema := new(textSchema)
	if source, ok := reflect.New(typ).Interface().(interface{ Descriptor() ([]byte, []int) }); ok {
		var descriptor *textMessageDescriptor
		var fullName string
		descriptor, fullName, schema.err = readTextMessageDescriptor(source)
		if schema.err == nil {
			schema.initNames(descriptor, fullName, GetProperties(typ))
		}
	}
	actual, _ := textSchemas.LoadOrStore(typ, schema)
	return actual.(*textSchema)
}

func readTextMessageDescriptor(source interface{ Descriptor() ([]byte, []int) }) (*textMessageDescriptor, string, error) {
	wire, path := source.Descriptor()
	reader, err := gzip.NewReader(bytes.NewReader(wire))
	if err != nil {
		return nil, "", err
	}
	defer reader.Close()
	// Generated descriptors are trusted schema data, but bound decompression so
	// a malformed hand-written Descriptor method cannot grow without limit.
	const maxDescriptor = 64 << 20
	data, err := ioutil.ReadAll(io.LimitReader(reader, maxDescriptor+1))
	if err != nil {
		return nil, "", err
	}
	if len(data) > maxDescriptor {
		return nil, "", fmt.Errorf("text descriptor too large")
	}
	var file textFileDescriptor
	if err := Unmarshal(data, &file); err != nil {
		return nil, "", err
	}
	messages := file.Messages
	var selected *textMessageDescriptor
	fullName := file.Package
	for _, index := range path {
		if index < 0 || index >= len(messages) {
			return nil, "", fmt.Errorf("invalid text descriptor path")
		}
		selected = messages[index]
		if fullName != "" {
			fullName += "."
		}
		fullName += selected.Name
		messages = selected.Nested
	}
	if selected == nil {
		return nil, "", fmt.Errorf("empty text descriptor path")
	}
	return selected, fullName, nil
}

func (schema *textSchema) initNames(descriptor *textMessageDescriptor, fullName string, properties *StructProperties) {
	schema.reserved = make(map[string]bool, len(descriptor.Reserved))
	for _, name := range descriptor.Reserved {
		schema.reserved[name] = true
	}
	schema.tagsByName = make(map[string]int, len(descriptor.Fields))
	schema.groupNames = make(map[int]string)
	for _, field := range descriptor.Fields {
		schema.tagsByName[field.Name] = int(field.Number)
	}
	for _, field := range descriptor.Fields {
		_, property, _ := textPropertyByTag(properties, int(field.Number))
		if property == nil || property.Wire != "group" {
			continue
		}
		name := field.Name
		// Group-like DELIMITED messages must be declared inside this message
		// and their field name must be the lowercased message name. Same-named
		// imported or enclosing messages do not gain group aliases.
		for _, nested := range descriptor.Nested {
			if (field.Type == 10 || field.Type == 11) && field.TypeName == "."+fullName+"."+nested.Name && field.Name == strings.ToLower(nested.Name) {
				if existingTag, existing := schema.tagsByName[nested.Name]; !existing || existingTag == int(field.Number) {
					name = nested.Name
					schema.tagsByName[nested.Name] = int(field.Number)
				}
				break
			}
		}
		schema.groupNames[int(field.Number)] = name
	}
}

func textPropertyByTag(properties *StructProperties, tag int) (int, *Properties, *OneofProperties) {
	if index, ok := properties.decoderTags.get(tag); ok {
		return index, properties.Prop[index], nil
	}
	for _, oneof := range properties.OneofTypes {
		if oneof.Prop.Tag == tag {
			return -1, oneof.Prop, oneof
		}
	}
	return -1, nil, nil
}

func writeStandardTextName(w *textWriter, properties *Properties, schema *textSchema) error {
	if schema != nil {
		if name, ok := schema.groupNames[properties.Tag]; ok {
			copy := *properties
			copy.OrigName = name
			return writeName(w, &copy)
		}
	}
	return writeName(w, properties)
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
	if depth > 100 || p.depth+depth > p.recursionLimit {
		return p.errorf("reserved text value nesting exceeds 100")
	}
	tok := p.next()
	if tok.err != nil {
		return tok.err
	}
	value := tok.value
	switch value {
	case "{", "<":
		if p.depth+depth >= p.recursionLimit {
			return p.errorf("reserved text message nesting limit")
		}
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
	if depth > 100 || p.depth+depth > p.recursionLimit {
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
