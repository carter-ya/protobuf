package proto

import (
	"bytes"
	"testing"
)

func TestEditionFieldBoundaries(t *testing.T) {
	good := [][]byte{{8, 0}, {17, 0, 0, 0, 0, 0, 0, 0, 0}, {26, 0}, {35, 43, 8, 1, 44, 36}, {45, 0, 0, 0, 0}}
	for _, data := range good {
		n, err := EditionFieldSize(append(append([]byte{}, data...), 8, 1))
		if err != nil || n != len(data) {
			t.Fatalf("%x: %d %v", data, n, err)
		}
	}
	bad := [][]byte{{}, {0}, {8}, {15}, {14}, {12}, {11, 20}, {11, 8, 1}, {18, 2, 1}, {17, 0}, {8, 128, 128, 128, 128, 128, 128, 128, 128, 128, 2}, {128, 128, 128, 128, 128, 1, 0}}
	for _, data := range bad {
		if _, err := EditionFieldSize(data); err == nil {
			t.Errorf("accepted %x", data)
		}
	}
	for _, data := range good {
		for n := 0; n < len(data); n++ {
			if _, err := EditionFieldSize(data[:n]); err == nil {
				t.Errorf("accepted truncated %x", data[:n])
			}
		}
	}
}
func TestGeneratedEditionEnumFilter(t *testing.T) {
	rules := map[uint64]*EditionEnumFilter{1: NewEditionEnumFilter([]int32{0, 1, -1}, false, true), 2: NewEditionEnumFilter([]int32{1}, true, false)}
	input := []byte{8, 1, 10, 3, 0, 9, 1, 8, 9, 18, 4, 8, 3, 16, 9, 24, 5}
	known, unknown, err := FilterEditionEnums(input, rules)
	if err != nil || !bytes.Equal(known, []byte{8, 1, 10, 2, 0, 1, 24, 5}) || !bytes.Equal(unknown, []byte{8, 9, 8, 9, 18, 4, 8, 3, 16, 9}) {
		t.Fatalf("%x %x %v", known, unknown, err)
	}
	clean := []byte{8, 1, 24, 2}
	same, extra, err := FilterEditionEnums(clean, rules)
	if err != nil || len(extra) != 0 || &same[0] != &clean[0] {
		t.Fatal("unmodified input copied")
	}
	for _, data := range [][]byte{{10, 1, 128}, {18, 2, 16}, {0}, {8, 128, 128, 128, 128, 128, 128, 128, 128, 128, 2}} {
		if _, _, err := FilterEditionEnums(data, rules); err == nil {
			t.Errorf("accepted malformed %x", data)
		}
	}
}
