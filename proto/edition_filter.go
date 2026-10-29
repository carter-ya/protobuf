package proto

import (
	"fmt"
	"io"
)

// EditionEnumFilter is immutable metadata for a generated closed-enum field.
// It is created once during package initialization, without runtime reflection.
type EditionEnumFilter struct{ filter closedEnumFilter }

func NewEditionEnumFilter(values []int32, mapValue, packed bool) *EditionEnumFilter {
	known := make(map[int32]bool, len(values))
	for _, value := range values {
		known[value] = true
	}
	return &EditionEnumFilter{filter: closedEnumFilter{values: known, mapValue: mapValue, allowPacked: packed, strictMap: mapValue}}
}

// FilterEditionEnums routes unknown closed-enum values before generated decoding.
// Unchanged input is returned without a copy. Unknown packed values become
// individual unknown varints; unknown scalar fields retain their entire wire.
func FilterEditionEnums(data []byte, fields map[uint64]*EditionEnumFilter) ([]byte, []byte, error) {
	var filtered, unknown []byte
	changed := false
	for offset := 0; offset < len(data); {
		start := offset
		key, n := decodeVarint(data[offset:])
		if n == 0 {
			return nil, nil, io.ErrUnexpectedEOF
		}
		offset += n
		tag, wire := key>>3, int(key&7)
		if n > 5 || tag == 0 || tag > 536870911 || wire == WireEndGroup {
			return nil, nil, fmt.Errorf("proto: invalid Editions field tag")
		}
		size, err := EditionFieldSize(data[start:])
		if err != nil {
			return nil, nil, err
		}
		end := start + size
		field := data[start:end]
		if rule := fields[tag]; rule != nil {
			body, extra, err := rule.filter.filter(data[offset:end], wire, tag)
			if err != nil && err != errInternalBadWireType {
				return nil, nil, err
			}
			if err == errInternalBadWireType || len(extra) > 0 {
				if !changed {
					filtered = append(filtered, data[:start]...)
					changed = true
				}
				if err == errInternalBadWireType {
					unknown = append(unknown, field...)
				} else {
					filtered = append(filtered, data[start:offset]...)
					filtered = append(filtered, body...)
					unknown = append(unknown, extra...)
				}
			} else if changed {
				filtered = append(filtered, field...)
			}
		} else if changed {
			filtered = append(filtered, field...)
		}
		offset = end
	}
	if !changed {
		return data, nil, nil
	}
	return filtered, unknown, nil
}

// NewEditionMapEnumFilter adds the map key validation policy to a closed enum.
func NewEditionMapEnumFilter(values []int32, verifyKey bool) *EditionEnumFilter {
	rule := NewEditionEnumFilter(values, true, false)
	rule.filter.verifyMapKey = verifyKey
	return rule
}
