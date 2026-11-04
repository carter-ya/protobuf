package proto

import (
	"bytes"
	"sync"
	"testing"
)

type extensionUTF8Probe struct {
	XXX_InternalExtensions
	XXX_unrecognized []byte
}

func (m *extensionUTF8Probe) Reset()       { *m = extensionUTF8Probe{} }
func (*extensionUTF8Probe) String() string { return "extensionUTF8Probe" }
func (*extensionUTF8Probe) ProtoMessage()  {}
func (*extensionUTF8Probe) ExtensionRangeArray() []ExtensionRange {
	return []ExtensionRange{{Start: 100, End: 110}}
}

var checkedStringExtension = &ExtensionDesc{ExtendedType: (*extensionUTF8Probe)(nil), ExtensionType: (*string)(nil), Field: 100, Name: "probe.checked", Tag: "bytes,100,opt,name=checked,proto3,explicit"}
var checkedStringsExtension = &ExtensionDesc{ExtendedType: (*extensionUTF8Probe)(nil), ExtensionType: ([]string)(nil), Field: 101, Name: "probe.checked_many", Tag: "bytes,101,rep,name=checked_many,proto3"}
var uncheckedStringExtension = &ExtensionDesc{ExtendedType: (*extensionUTF8Probe)(nil), ExtensionType: (*string)(nil), Field: 102, Name: "probe.unchecked", Tag: "bytes,102,opt,name=unchecked,proto3,explicit,utf8_none"}
var legacyStringExtension = &ExtensionDesc{ExtendedType: (*extensionUTF8Probe)(nil), ExtensionType: (*string)(nil), Field: 103, Name: "probe.legacy", Tag: "bytes,103,opt,name=legacy"}
var lateStringExtension = &ExtensionDesc{ExtendedType: (*extensionUTF8Probe)(nil), ExtensionType: (*string)(nil), Field: 104, Name: "probe.late", Tag: "bytes,104,opt,name=late,proto3,explicit"}
var registerLateUTF8 sync.Once

func init() {
	for _, desc := range []*ExtensionDesc{checkedStringExtension, checkedStringsExtension, uncheckedStringExtension, legacyStringExtension} {
		RegisterExtension(desc)
	}
}
func extensionStringWire(field int, value []byte) []byte {
	wire := EncodeVarint(uint64(field<<3 | WireBytes))
	wire = append(wire, EncodeVarint(uint64(len(value)))...)
	return append(wire, value...)
}
func TestRegisteredExtensionUTF8Validation(t *testing.T) {
	for _, desc := range []*ExtensionDesc{checkedStringExtension, checkedStringsExtension} {
		for _, value := range [][]byte{nil, []byte("snow 雪"), {255}} {
			wire := extensionStringWire(int(desc.Field), value)
			var message extensionUTF8Probe
			err := Unmarshal(wire, &message)
			if bytes.Equal(value, []byte{255}) {
				if err == nil {
					t.Fatalf("accepted invalid UTF-8 extension %s", desc.Name)
				}
			} else if err != nil {
				t.Fatalf("valid extension %s: %v", desc.Name, err)
			}
		}
	}
	for _, desc := range []*ExtensionDesc{uncheckedStringExtension, legacyStringExtension} {
		var message extensionUTF8Probe
		if err := Unmarshal(extensionStringWire(int(desc.Field), []byte{255}), &message); err != nil {
			t.Fatalf("changed unchecked/legacy extension: %v", err)
		}
		if _, err := GetExtension(&message, desc); err != nil {
			t.Fatal(err)
		}
	}
	// Registration after the first table use must not bypass validation.
	registerLateUTF8.Do(func() { RegisterExtension(lateStringExtension) })
	if err := Unmarshal(extensionStringWire(104, []byte{255}), new(extensionUTF8Probe)); err == nil {
		t.Fatal("late registered extension bypassed UTF-8 validation")
	}
	// A known string with another wire type is an unknown field, as for a
	// normal string field; it must not acquire extension presence.
	badType := append(EncodeVarint(100<<3), 1)
	var message extensionUTF8Probe
	if err := Unmarshal(badType, &message); err != nil || HasExtension(&message, checkedStringExtension) || !bytes.Equal(message.XXX_unrecognized, badType) {
		t.Fatalf("wrong-wire routing: %v", err)
	}
}
