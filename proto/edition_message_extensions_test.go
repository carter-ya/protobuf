package proto

import (
	"bytes"
	"testing"
	"time"
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
	Text  string                       `protobuf:"bytes,2,opt,name=text,proto3"`
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

func TestEditionExtensionFragmentBoundaries(t *testing.T) {
	for _, input := range [][]byte{
		append(extensionWire(100, []byte{26, 3, 'A'}), extensionWire(100, []byte{8, 1})...),
		append(extensionWire(110, []byte{10, 4, 8, 1}), extensionWire(110, []byte{16, 2})...),
	} {
		var got extensionValidationRecord
		if err := Unmarshal(input, &got); err == nil {
			t.Fatalf("truncated occurrence consumed following fragment: %x", input)
		}
		number := int32(100)
		extension := validationPayload
		if input[0] != extensionWire(100, nil)[0] {
			number = 110
			extension = validationNested
		}
		SetRawExtension(&got, number, input)
		if _, err := GetExtension(&got, extension); err == nil {
			t.Fatal("GetExtension accepted truncated fragment")
		}
	}
}
func TestEditionRequiredRecoveryKeepsUTF8Errors(t *testing.T) {
	body := []byte{10, 2, 8, 1, 18, 1, 255, 10, 2, 16, 2, 18, 1, 'x'}
	input := extensionWire(110, body)
	var got extensionValidationRecord
	if err := Unmarshal(input, &got); err == nil {
		t.Fatal("required recovery hid an overwritten UTF-8 error")
	}
	SetRawExtension(&got, 110, input)
	if _, err := GetExtension(&got, validationNested); err == nil {
		t.Fatal("GetExtension hid an overwritten UTF-8 error")
	}
}

type extensionValidationMap struct {
	Values map[string]*extensionValidationRequired `protobuf:"bytes,1,rep,name=values,proto3" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value"`
}

func (m *extensionValidationMap) Reset()         { *m = extensionValidationMap{} }
func (m *extensionValidationMap) String() string { return CompactTextString(m) }
func (*extensionValidationMap) ProtoMessage()    {}

var validationMap = &ExtensionDesc{ExtendedType: (*extensionValidationRecord)(nil), ExtensionType: (*extensionValidationMap)(nil), Field: 111, Name: "validation.map", Tag: "bytes,111,edition_ext,opt,name=map,proto3,explicit"}

func init() { RegisterExtension(validationMap) }
func TestEditionRequiredMapRecoveryKeepsUTF8Errors(t *testing.T) {
	entry := []byte{18, 2, 8, 1, 10, 1, 255, 18, 2, 16, 2, 10, 1, 'x'}
	input := extensionWire(111, extensionWire(1, entry))
	var got extensionValidationRecord
	if err := Unmarshal(input, &got); err == nil {
		t.Fatal("map accumulator masked overwritten UTF-8 error with required error")
	}
	SetRawExtension(&got, 111, input)
	if _, err := GetExtension(&got, validationMap); err == nil {
		t.Fatal("map GetExtension masked UTF-8 error")
	}
}

func TestEditionRequiredMapDefaultValue(t *testing.T) {
	input := extensionWire(111, extensionWire(1, []byte{10, 1, 'x'}))
	var got extensionValidationRecord
	if err := Unmarshal(input, &got); err == nil {
		t.Fatal("omitted map message value bypassed required checks")
	}
	SetRawExtension(&got, 111, input)
	if _, err := GetExtension(&got, validationMap); err == nil {
		t.Fatal("map default GetExtension bypassed required checks")
	}
}

var validationRecursive = &ExtensionDesc{ExtendedType: (*extensionValidationRecord)(nil), ExtensionType: (*extensionValidationRecord)(nil), Field: 112, Name: "validation.recursive", Tag: "bytes,112,opt,edition_ext,name=recursive,proto3,explicit"}

func init() { RegisterExtension(validationRecursive) }
func TestEditionRecursiveExtensionWork(t *testing.T) {
	for _, depth := range []int{12, 16, 64} {
		var input []byte
		for i := 0; i < depth; i++ {
			input = append(extensionWire(112, input), extensionWire(112, nil)...)
		}
		done := make(chan error, 1)
		start := time.Now()
		go func() { done <- Unmarshal(input, new(extensionValidationRecord)) }()
		select {
		case err := <-done:
			if err != nil {
				t.Fatal(err)
			}
		case <-time.After(5 * time.Second):
			t.Fatalf("recursive extension decoding did not finish at depth %d", depth)
		}
		t.Logf("depth=%d bytes=%d elapsed=%v", depth, len(input), time.Since(start))
	}
}
func TestEditionEmptyRepeatedExtensionMerge(t *testing.T) {
	var message extensionValidationRecord
	if err := SetExtension(&message, validationRepeated, []*extensionValidationRequired{}); err != nil {
		t.Fatal(err)
	}
	if err := UnmarshalMerge(nil, &message); err != nil {
		t.Fatal(err)
	}
	value, err := GetExtension(&message, validationRepeated)
	if err != nil || len(value.([]*extensionValidationRequired)) != 0 {
		t.Fatalf("empty repeated extension: %v", err)
	}
}

func TestEditionNestedExtensionRequiredFragments(t *testing.T) {
	partial := extensionWire(112, extensionWire(104, []byte{8, 1}))
	complete := extensionWire(112, extensionWire(104, []byte{16, 2}))
	if err := Unmarshal(append(partial, complete...), new(extensionValidationRecord)); err != nil {
		t.Fatal(err)
	}
	if err := Unmarshal(partial, new(extensionValidationRecord)); err == nil {
		t.Fatal("nested extension required fields were not checked")
	}
	bad := extensionWire(112, append(extensionWire(100, []byte{26, 3, 'A'}), extensionWire(100, []byte{8, 1})...))
	if err := Unmarshal(bad, new(extensionValidationRecord)); err == nil {
		t.Fatal("nested extension consumed following occurrence")
	}
}
