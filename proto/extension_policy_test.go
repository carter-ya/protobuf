package proto

import "testing"

var validationOldProto3 = &ExtensionDesc{ExtendedType: (*extensionValidationRecord)(nil), ExtensionType: (*extensionValidationPayload)(nil), Field: 108, Name: "validation.old_proto3", Tag: "bytes,108,opt,name=old_proto3,proto3,explicit"}
var validationScalar = &ExtensionDesc{ExtendedType: (*extensionValidationRecord)(nil), ExtensionType: (*int32)(nil), Field: 109, Name: "validation.scalar", Tag: "varint,109,edition_ext,opt,name=scalar,proto3,explicit"}

func init() { RegisterExtension(validationOldProto3); RegisterExtension(validationScalar) }
func TestExtensionPolicyIsExplicitMetadata(t *testing.T) {
	var old extensionValidationRecord
	if err := Unmarshal(extensionWire(108, []byte{8}), &old); err != nil {
		t.Fatalf("proto3 alone must not enable the new policy: %v", err)
	}
	if _, err := GetExtension(&old, validationOldProto3); err == nil {
		t.Fatal("old proto3 payload error must stay lazy")
	}
	var properties Properties
	properties.Parse("bytes,1,opt,name=text,def=prefix,edition_ext,suffix")
	if properties.editionExtension {
		t.Fatal("default string was mistaken for a policy marker")
	}
	properties.Parse("bytes,1,edition_ext,opt,name=text,proto3")
	if !properties.editionExtension {
		t.Fatal("missing explicit policy")
	}
	var roundtrip Properties
	roundtrip.Parse(properties.String())
	if !roundtrip.editionExtension {
		t.Fatal("policy lost on property roundtrip")
	}
	var got extensionValidationRecord
	if err := SetExtension(&got, validationScalar, Int32(1)); err != nil {
		t.Fatal(err)
	}
	raw := encodeVarint(encodeVarint(nil, 109<<3|WireVarint), 2)
	if err := UnmarshalMerge(raw, &got); err != nil {
		t.Fatal(err)
	}
	value, err := GetExtension(&got, validationScalar)
	if err != nil || *value.(*int32) != 2 {
		t.Fatalf("decoded scalar extension did not update: %v", err)
	}
}

type bytesValidationRecord struct {
	XXX_extensions   []byte
	XXX_unrecognized []byte
}

func (m *bytesValidationRecord) Reset()         { *m = bytesValidationRecord{} }
func (m *bytesValidationRecord) String() string { return CompactTextString(m) }
func (*bytesValidationRecord) ProtoMessage()    {}
func (*bytesValidationRecord) ExtensionRangeArray() []ExtensionRange {
	return []ExtensionRange{{100, 199}}
}
func (m *bytesValidationRecord) GetExtensions() *[]byte { return &m.XXX_extensions }

var bytesRequired = &ExtensionDesc{ExtendedType: (*bytesValidationRecord)(nil), ExtensionType: (*extensionValidationRequired)(nil), Field: 104, Name: "validation.bytes_required", Tag: "bytes,104,edition_ext,opt,name=required,proto3,explicit"}
var bytesRepeated = &ExtensionDesc{ExtendedType: (*bytesValidationRecord)(nil), ExtensionType: []int32(nil), Field: 105, Name: "validation.bytes_repeated", Tag: "varint,105,edition_ext,rep,name=repeated,proto3"}

func init() { RegisterExtension(bytesRequired); RegisterExtension(bytesRepeated) }
func TestEditionExtensionsWithByteStorage(t *testing.T) {
	input := append(extensionWire(104, []byte{8, 1}), extensionWire(104, []byte{16, 2})...)
	var got bytesValidationRecord
	if err := Unmarshal(input, &got); err != nil {
		t.Fatal(err)
	}
	value, err := GetExtension(&got, bytesRequired)
	if err != nil {
		t.Fatal(err)
	}
	payload := value.(*extensionValidationRequired)
	if payload.A == nil || *payload.A != 1 || payload.B == nil || *payload.B != 2 {
		t.Fatal("byte storage lost required fragments")
	}
	raw := append(encodeVarint(encodeVarint(nil, 105<<3|WireVarint), 1), encodeVarint(encodeVarint(nil, 105<<3|WireVarint), 2)...)
	if err := Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}
	value, err = GetExtension(&got, bytesRepeated)
	if err != nil || len(value.([]int32)) != 2 {
		t.Fatalf("byte storage lost repeated fragments: %v", err)
	}
	if err := Unmarshal(extensionWire(104, []byte{8}), &got); err == nil {
		t.Fatal("byte storage accepted malformed registered payload")
	}
}

type lateValidationRecord struct {
	XXX_InternalExtensions
	XXX_unrecognized []byte
}

func (m *lateValidationRecord) Reset()         { *m = lateValidationRecord{} }
func (m *lateValidationRecord) String() string { return CompactTextString(m) }
func (*lateValidationRecord) ProtoMessage()    {}
func (*lateValidationRecord) ExtensionRangeArray() []ExtensionRange {
	return []ExtensionRange{{100, 199}}
}
func TestEditionExtensionRegistrationAfterTableInitialization(t *testing.T) {
	var got lateValidationRecord
	if err := Unmarshal(nil, &got); err != nil {
		t.Fatal(err)
	}
	if RegisteredExtensions(&got)[100] == nil {
		RegisterExtension(&ExtensionDesc{ExtendedType: (*lateValidationRecord)(nil), ExtensionType: (*extensionValidationPayload)(nil), Field: 100, Name: "validation.late", Tag: "bytes,100,edition_ext,opt,name=late,proto3,explicit"})
	}
	SetRawExtension(&got, 100, extensionWire(100, []byte{8}))
	if err := UnmarshalMerge(nil, &got); err == nil {
		t.Fatal("cached table ignored late registration")
	}
}
