package legacy_test

import (
	"bytes"
	old "example.com/gogo-wkt-legacy/types"
	"github.com/gogo/protobuf/proto"
	"testing"
)

func TestFrozenWKTConsumers(t *testing.T) {
	for _, tc := range []struct {
		value  proto.Message
		newTag byte
	}{
		{&old.Type{Name: "legacy", Syntax: old.Syntax(2)}, 0x3a},
		{&old.Enum{Name: "legacy", Syntax: old.Syntax(2)}, 0x32},
		{&old.Api{Name: "legacy", Syntax: old.Syntax(2)}, 0x42},
		{&old.Method{Name: "legacy", Syntax: old.Syntax(2)}, 0x42},
	} {
		wire, err := proto.Marshal(tc.value)
		if err != nil {
			t.Fatal(err)
		}
		wire = append(wire, tc.newTag, 4, '2', '0', '2', '6')
		tc.value.Reset()
		if err := proto.Unmarshal(wire, tc.value); err != nil {
			t.Fatal(err)
		}
		back, err := proto.Marshal(tc.value)
		if err != nil || !bytes.Equal(wire, back) {
			t.Fatalf("frozen %T lost unknown edition: %x %v", tc.value, back, err)
		}
	}
	if old.Syntax(2).String() != "2" {
		t.Fatal("frozen enum API unexpectedly changed")
	}
}
