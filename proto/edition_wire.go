package proto

import (
	"fmt"
	"io"
)

func editionVarint(data []byte) (uint64, int, error) {
	var value uint64
	for index, b := range data {
		if index == 10 || (index == 9 && b > 1) {
			return 0, 0, fmt.Errorf("proto: Editions varint overflow")
		}
		value |= uint64(b&127) << uint(index*7)
		if b < 128 {
			return value, index + 1, nil
		}
	}
	return 0, 0, io.ErrUnexpectedEOF
}

// EditionFieldSize bounds one complete field, including matching group tags.
// Generated Editions codecs use this for unknown/wrong-wire fields. It does not
// change the historical decoder's acceptance behavior.
func EditionFieldSize(data []byte) (int, error) {
	end, _, _, err := editionField(data)
	return end, err
}
func editionField(data []byte) (int, int, int, error) {
	offset := 0
	firstHeader := 0
	var groups []uint64
	for offset < len(data) {
		headerStart := offset
		key, n, err := editionVarint(data[offset:])
		if err != nil {
			return 0, 0, 0, err
		}
		tag, wire := key>>3, int(key&7)
		if n > 5 || tag == 0 || tag > 536870911 {
			return 0, 0, 0, fmt.Errorf("proto: invalid Editions field tag")
		}
		offset += n
		if firstHeader == 0 {
			firstHeader = offset
		}
		switch wire {
		case WireVarint:
			_, size, err := editionVarint(data[offset:])
			if err != nil {
				return 0, 0, 0, err
			}
			offset += size
		case WireFixed64:
			if len(data)-offset < 8 {
				return 0, 0, 0, io.ErrUnexpectedEOF
			}
			offset += 8
		case WireFixed32:
			if len(data)-offset < 4 {
				return 0, 0, 0, io.ErrUnexpectedEOF
			}
			offset += 4
		case WireBytes:
			length, size, err := editionVarint(data[offset:])
			if err != nil {
				return 0, 0, 0, err
			}
			offset += size
			if length > uint64(len(data)-offset) {
				return 0, 0, 0, io.ErrUnexpectedEOF
			}
			offset += int(length)
		case WireStartGroup:
			if len(groups) >= 10000 {
				return 0, 0, 0, fmt.Errorf("proto: Editions group nesting limit")
			}
			groups = append(groups, tag)
			continue
		case WireEndGroup:
			if len(groups) == 0 || groups[len(groups)-1] != tag {
				return 0, 0, 0, fmt.Errorf("proto: mismatched Editions group tag")
			}
			groups = groups[:len(groups)-1]
			if len(groups) == 0 {
				return offset, firstHeader, headerStart, nil
			}
			continue
		default:
			return 0, 0, 0, fmt.Errorf("proto: invalid Editions wire type")
		}
		if len(groups) == 0 {
			return offset, firstHeader, offset, nil
		}
	}
	return 0, 0, 0, io.ErrUnexpectedEOF
}
