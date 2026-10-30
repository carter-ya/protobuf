package proto

import (
	"fmt"
	"unicode/utf8"
)

// FilterEditionExtensions validates registered declaration metadata before raw
// extension bytes enter the lazy extension store. Unknown registrations retain
// their original wire and do not require descriptors or V2 reflection.
func FilterEditionExtensions(data []byte, message Message) ([]byte, []byte, error) {
	rules := make(map[uint64]*EditionEnumFilter)
	extensions := RegisteredExtensions(message)
	for number, extension := range extensions {
		p := extensionProperties(extension)
		if len(p.ClosedEnum) > 0 {
			rules[uint64(number)] = &EditionEnumFilter{filter: closedEnumFilter{values: p.ClosedEnum, allowPacked: p.Repeated}}
		}
	}
	var filtered, unknown []byte
	changed := false
	for offset := 0; offset < len(data); {
		key, n, err := editionVarint(data[offset:])
		if err != nil {
			return nil, nil, err
		}
		size, err := EditionFieldSize(data[offset:])
		if err != nil {
			return nil, nil, err
		}
		if extension := extensions[int32(key>>3)]; extension != nil {
			p := extensionProperties(extension)
			expected := p.WireType
			if p.Wire == "group" {
				expected = WireStartGroup
			}
			wire := int(key & 7)
			packed := p.Repeated && (expected == WireVarint || expected == WireFixed32 || expected == WireFixed64) && wire == WireBytes
			if wire != expected && !packed {
				if !changed {
					filtered = append(filtered, data[:offset]...)
					changed = true
				}
				unknown = append(unknown, data[offset:offset+size]...)
				offset += size
				continue
			}
			// The type, not the bytes wire kind, distinguishes strings from bytes.
			if p.proto3 && !p.UTF8None && key&7 == WireBytes {
				switch extension.ExtensionType.(type) {
				case *string, []string:
					length, prefix, err := editionVarint(data[offset+n : offset+size])
					if err != nil {
						return nil, nil, err
					}
					if !utf8.Valid(data[offset+n+prefix : offset+n+prefix+int(length)]) {
						return nil, nil, fmt.Errorf("proto: invalid UTF-8 in Editions extension %s", extension.Name)
					}
				}
			}
		}
		if changed {
			filtered = append(filtered, data[offset:offset+size]...)
		}
		offset += size
	}
	if changed {
		data = filtered
	}
	known, extra, err := FilterEditionEnums(data, rules)
	return known, append(unknown, extra...), err
}
