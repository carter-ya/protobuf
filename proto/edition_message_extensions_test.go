package proto

import (
	"bytes"
	"testing"
)

type extensionValidationPayload struct {
	A    *int32 `protobuf:"varint,1,opt,name=a"`
	B    *int32 `protobuf:"varint,2,opt,name=b"`
	Text string `protobuf:"bytes,3,opt,name=text,proto3"`
}

func (m *extensionValidationPayload) Reset()         { *m = extensionValidationPayload{} }
func (m *extensionValidationPayload) String() string { return CompactTextString(m) }
func (*extensionValidationPayload) ProtoMessage()    {}

type extensionValidationRequired struct {
	A *int32 `protobuf:"varint,1,req,name=a"`
	B *int32 `protobuf:"varint,2,req,name=b"`
}

func (m *extensionValidationRequired) Reset()         { *m = extensionValidationRequired{} }
func (m *extensionValidationRequired) String() string { return CompactTextString(m) }
func (*extensionValidationRequired) ProtoMessage()    {}

type extensionValidationRecord struct {
	XXX_InternalExtensions
	XXX_unrecognized []byte
}

func (m *extensionValidationRecord) Reset()         { *m = extensionValidationRecord{} }
func (m *extensionValidationRecord) String() string { return CompactTextString(m) }
func (*extensionValidationRecord) ProtoMessage()    {}
func (*extensionValidationRecord) ExtensionRangeArray() []ExtensionRange {
	return []ExtensionRange{{100, 199}}
}

var validationPayload = &ExtensionDesc{ExtendedType: (*extensionValidationRecord)(nil), ExtensionType: (*extensionValidationPayload)(nil), Field: 100, Name: "validation.payload", Tag: "bytes,100,opt,edition_ext,name=payload,proto3,explicit"}
var validationRequired = &ExtensionDesc{ExtendedType: (*extensionValidationRecord)(nil), ExtensionType: (*extensionValidationRequired)(nil), Field: 104, Name: "validation.required", Tag: "bytes,104,opt,edition_ext,name=required,proto3,explicit"}
var validationRepeated = &ExtensionDesc{ExtendedType: (*extensionValidationRecord)(nil), ExtensionType: []*extensionValidationRequired(nil), Field: 105, Name: "validation.repeated", Tag: "bytes,105,rep,edition_ext,name=repeated,proto3"}
var validationGroup = &ExtensionDesc{ExtendedType: (*extensionValidationRecord)(nil), ExtensionType: (*extensionValidationPayload)(nil), Field: 106, Name: "validation.group", Tag: "group,106,opt,edition_ext,name=group,proto3,explicit"}
var validationLegacy = &ExtensionDesc{ExtendedType: (*extensionValidationRecord)(nil), ExtensionType: (*extensionValidationPayload)(nil), Field: 107, Name: "validation.legacy", Tag: "bytes,107,opt,name=legacy"}

func init() {
	for _, extension := range []*ExtensionDesc{validationPayload, validationRequired, validationRepeated, validationGroup, validationLegacy} {
		RegisterExtension(extension)
	}
}
func extensionWire(number int32, body []byte) []byte {
	data := encodeVarint(nil, uint64(number)<<3|WireBytes)
	data = encodeVarint(data, uint64(len(body)))
	return append(data, body...)
}
func TestEditionNativeExtensionValidation(t *testing.T) {
	for _, input := range [][]byte{
		extensionWire(100, []byte{8}), extensionWire(100, []byte{0}), extensionWire(100, []byte{26, 1, 255}),
		extensionWire(104, nil), extensionWire(104, []byte{8, 1}), extensionWire(105, []byte{8, 1}),
		append(append(encodeVarint(nil, 106<<3|WireStartGroup), 0), encodeVarint(nil, 106<<3|WireEndGroup)...),
	} {
		if err := Unmarshal(input, new(extensionValidationRecord)); err == nil {
			t.Fatalf("accepted invalid registered extension %x", input)
		}
	}
	var legacy extensionValidationRecord
	if err := Unmarshal(extensionWire(107, []byte{8}), &legacy); err != nil {
		t.Fatalf("legacy lazy policy changed: %v", err)
	}
	if _, err := GetExtension(&legacy, validationLegacy); err == nil {
		t.Fatal("legacy payload error must remain deferred")
	}
	unknown := extensionWire(150, []byte{0})
	var got extensionValidationRecord
	if err := Unmarshal(unknown, &got); err != nil {
		t.Fatal("unknown message-like bytes must stay opaque", err)
	}
	output, err := Marshal(&got)
	if err != nil || !bytes.Equal(output, unknown) {
		t.Fatalf("unknown extension changed: %x %v", output, err)
	}
	wrong := encodeVarint(encodeVarint(nil, 100<<3|WireVarint), 1)
	if err := Unmarshal(wrong, &got); err != nil || HasExtension(&got, validationPayload) || !bytes.Equal(got.XXX_unrecognized, wrong) {
		t.Fatalf("wrong wire must be unknown: %v", err)
	}
}
func TestEditionRequiredExtensionFragments(t *testing.T) {
	input := append(extensionWire(104, []byte{8, 1}), extensionWire(104, []byte{16, 2})...)
	var got extensionValidationRecord
	if err := Unmarshal(input, &got); err != nil {
		t.Fatal(err)
	}
	value, err := GetExtension(&got, validationRequired)
	if err != nil {
		t.Fatal(err)
	}
	payload := value.(*extensionValidationRequired)
	if payload.A == nil || *payload.A != 1 || payload.B == nil || *payload.B != 2 {
		t.Fatal("required fragments did not merge")
	}
	repeated := append(extensionWire(105, []byte{8, 1, 16, 2}), extensionWire(105, []byte{8, 3, 16, 4})...)
	if err := Unmarshal(repeated, &got); err != nil {
		t.Fatal(err)
	}
	value, err = GetExtension(&got, validationRepeated)
	if err != nil || len(value.([]*extensionValidationRequired)) != 2 {
		t.Fatalf("repeated extension changed: %v", err)
	}
	group := append(append(encodeVarint(nil, 106<<3|WireStartGroup), 8, 1), encodeVarint(nil, 106<<3|WireEndGroup)...)
	if err := Unmarshal(group, &got); err != nil {
		t.Fatal(err)
	}
	value, err = GetExtension(&got, validationGroup)
	if err != nil || *value.(*extensionValidationPayload).A != 1 {
		t.Fatalf("group extension changed: %v", err)
	}
}
func TestEditionDecodedExtensionMergeAndLazyStore(t *testing.T) {
	for _, set := range []bool{false, true} {
		var got extensionValidationRecord
		if set {
			if err := SetExtension(&got, validationPayload, &extensionValidationPayload{A: Int32(1)}); err != nil {
				t.Fatal(err)
			}
		} else {
			if err := Unmarshal(extensionWire(100, []byte{8, 1}), &got); err != nil {
				t.Fatal(err)
			}
			values, lock := got.extensionsRead()
			lock.Lock()
			value := values[100]
			lock.Unlock()
			if value.value != nil || len(value.enc) == 0 {
				t.Fatal("validation must retain a lazy store")
			}
			if _, err := GetExtension(&got, validationPayload); err != nil {
				t.Fatal(err)
			}
		}
		if err := UnmarshalMerge(extensionWire(100, []byte{16, 2}), &got); err != nil {
			t.Fatal(err)
		}
		value, err := GetExtension(&got, validationPayload)
		if err != nil {
			t.Fatal(err)
		}
		payload := value.(*extensionValidationPayload)
		if payload.A == nil || *payload.A != 1 || payload.B == nil || *payload.B != 2 {
			t.Fatal("decoded extension merge lost fields")
		}
	}
}

type extensionValidationNested struct {
	Child *extensionValidationRequired `protobuf:"bytes,1,opt,name=child"`
}

func (m *extensionValidationNested) Reset()         { *m = extensionValidationNested{} }
func (m *extensionValidationNested) String() string { return CompactTextString(m) }
func (*extensionValidationNested) ProtoMessage()    {}

var validationNested = &ExtensionDesc{ExtendedType: (*extensionValidationRecord)(nil), ExtensionType: (*extensionValidationNested)(nil), Field: 110, Name: "validation.nested", Tag: "bytes,110,edition_ext,opt,name=nested,proto3,explicit"}

func init() { RegisterExtension(validationNested) }
func TestEditionNestedRequiredExtensionFragments(t *testing.T) {
	input := append(extensionWire(110, []byte{10, 2, 8, 1}), extensionWire(110, []byte{10, 2, 16, 2})...)
	var got extensionValidationRecord
	if err := Unmarshal(input, &got); err != nil {
		t.Fatal("nested required fragments must be checked after merging", err)
	}
	value, err := GetExtension(&got, validationNested)
	if err != nil || value.(*extensionValidationNested).Child.A == nil || value.(*extensionValidationNested).Child.B == nil {
		t.Fatalf("nested merge lost fields: %v", err)
	}
	if err := SetExtension(&got, validationRequired, &extensionValidationRequired{A: Int32(1)}); err != nil {
		t.Fatal(err)
	}
	if err := UnmarshalMerge(extensionWire(104, []byte{16, 2}), &got); err == nil {
		t.Fatal("invalid decoded prior value must return an encoding error")
	}
	value, err = GetExtension(&got, validationRequired)
	if err != nil || value.(*extensionValidationRequired).A == nil || *value.(*extensionValidationRequired).A != 1 {
		t.Fatal("encoding error lost prior cached fields")
	}
}
