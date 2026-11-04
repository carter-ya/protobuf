package proto

import "reflect"

// Standard maps are repeated entry messages and accept both list and repeated
// body notation. Keep the historical map parser independent of these rules.
func (p *textParser) readStandardMap(dst reflect.Value, properties *Properties) error {
	if err := p.checkForColon(properties, dst.Type()); err != nil {
		return err
	}
	token := p.next()
	if token.err != nil {
		return token.err
	}
	if token.value != "[" {
		p.back()
		if err := p.readStandardMapEntry(dst, properties); err != nil {
			return err
		}
		return p.consumeOptionalSeparator()
	}
	token = p.next()
	if token.err != nil {
		return token.err
	}
	if token.value == "]" {
		return p.consumeOptionalSeparator()
	}
	p.back()
	for {
		if err := p.readStandardMapEntry(dst, properties); err != nil {
			return err
		}
		token = p.next()
		if token.err != nil {
			return token.err
		}
		if token.value == "]" {
			break
		}
		if token.value != "," {
			return p.errorf("expected ',' or ']' in map list")
		}
	}
	return p.consumeOptionalSeparator()
}

func (p *textParser) readStandardMapEntry(dst reflect.Value, properties *Properties) error {
	if p.depth >= p.recursionLimit {
		return p.errorf("standard text map nesting exceeds limit %d", p.recursionLimit)
	}
	p.depth++
	defer func() { p.depth-- }()
	token := p.next()
	if token.err != nil {
		return token.err
	}
	end := "}"
	switch token.value {
	case "{":
	case "<":
		end = ">"
	default:
		return p.errorf("expected map entry body, found %q", token.value)
	}
	key := reflect.New(dst.Type().Key()).Elem()
	value := reflect.New(dst.Type().Elem()).Elem()
	keySeen, valueSeen := false, false
	for {
		token = p.next()
		if token.err != nil {
			return token.err
		}
		if token.value == end {
			break
		}
		switch token.value {
		case "key":
			if keySeen {
				return p.errorf("map entry key cannot be repeated")
			}
			keySeen = true
			if err := p.consumeToken(":"); err != nil {
				return err
			}
			if err := p.readAny(key, properties.MapKeyProp); err != nil {
				return err
			}
		case "value":
			if valueSeen {
				return p.errorf("map entry value cannot be repeated")
			}
			valueSeen = true
			if err := p.checkForColon(properties.MapValProp, value.Type()); err != nil {
				return err
			}
			if err := p.readAny(value, properties.MapValProp); err != nil {
				return err
			}
		default:
			return p.errorf("unknown map entry field %q", token.value)
		}
		if err := p.consumeOptionalSeparator(); err != nil {
			return err
		}
	}
	if dst.IsNil() {
		dst.Set(reflect.MakeMap(dst.Type()))
	}
	dst.SetMapIndex(key, value)
	return nil
}
