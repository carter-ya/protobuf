package editions_test

import (
	"bytes"
	"compress/gzip"
	"encoding/hex"
	"io/ioutil"
	"os"
	"os/exec"
	"reflect"
	"strings"
	"testing"

	pb "github.com/gogo/protobuf/compat/editions/edition2023"
	"github.com/gogo/protobuf/jsonpb"
	"github.com/gogo/protobuf/proto"
	d "github.com/gogo/protobuf/protoc-gen-gogo/descriptor"
)

func wire(s string) []byte {
	b, err := hex.DecodeString(s)
	if err != nil {
		panic(err)
	}
	return b
}
func marshal(t *testing.T, m proto.Message) []byte {
	t.Helper()
	b, err := proto.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	if proto.Size(m) != len(b) {
		t.Fatal("size mismatch")
	}
	return b
}

func TestPresenceAndUnknowns(t *testing.T) {
	m := &pb.Sample{Value: proto.Int32(0), ExplicitBytes: []byte{}}
	if b := marshal(t, m); !bytes.Equal(b, wire("08001200")) {
		t.Fatalf("explicit zeros: %x", b)
	}
	clone := proto.Clone(m).(*pb.Sample)
	if !proto.Equal(clone, m) || clone.ExplicitBytes == nil {
		t.Fatal("clone lost presence")
	}
	if proto.Equal(m, new(pb.Sample)) {
		t.Fatal("presence ignored by Equal")
	}
	dst := &pb.Sample{Value: proto.Int32(9), ExplicitBytes: []byte{1}, Implicit: 9}
	proto.Merge(dst, m)
	if dst.Value == nil || *dst.Value != 0 || dst.ExplicitBytes == nil || len(dst.ExplicitBytes) != 0 || dst.Implicit != 9 {
		t.Fatal("merge lost field semantics")
	}
	j, err := (&jsonpb.Marshaler{EmitDefaults: true}).MarshalToString(new(pb.Sample))
	if err != nil || strings.Contains(j, `"value"`) || strings.Contains(j, `"explicitBytes"`) {
		t.Fatalf("JSON absent explicit fields: %s %v", j, err)
	}
	if new(pb.Sample).GetImported() != pb.Imported_FIRST {
		t.Fatal("imported proto2 enum default must be first value, not zero")
	}
	for _, tc := range []struct {
		name, input, unknown string
		list                 []pb.Closed
	}{
		{"singular wrong wire", "4a020901", "4a020901", nil},
		{"oneof wrong wire", "8201020901", "8201020901", nil},
		{"singular", "4809", "4809", nil},
		{"packed", "520400090108", "50095008", []pb.Closed{0, 1}},
		{"expanded", "500050095001", "5009", []pb.Closed{0, 1}},
		{"map", "5a050a016b1009", "5a050a016b1009", nil},
		{"imported", "900103", "900103", nil},
		{"oneof", "800109", "800109", nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := new(pb.Sample)
			if err := proto.Unmarshal(wire(tc.input), got); err != nil {
				t.Fatal(err)
			}
			if got.Closed != nil || got.Imported != nil || got.Choice != nil || len(got.ClosedMap) != 0 || !reflect.DeepEqual(got.ClosedPacked, tc.list) || !bytes.Equal(got.XXX_unrecognized, wire(tc.unknown)) {
				t.Fatalf("closed enum decode: %+v", got)
			}
			if !bytes.HasSuffix(marshal(t, got), wire(tc.unknown)) {
				t.Fatal("unknown enum values lost")
			}
		})
	}
	got := &pb.Sample{Choice: &pb.Sample_SelectedInt{SelectedInt: 7}}
	if err := proto.UnmarshalMerge(wire("800109"), got); err != nil {
		t.Fatal(err)
	}
	if got.GetSelectedInt() != 7 {
		t.Fatal("unknown closed enum replaced existing oneof")
	}
	open := new(pb.Sample)
	if err := proto.Unmarshal(wire("6009"), open); err != nil || open.Open == nil || *open.Open != 9 {
		t.Fatalf("open enum rejected: %+v %v", open, err)
	}
	if err := jsonpb.UnmarshalString(`{"closed":9}`, new(pb.Sample)); err == nil {
		t.Fatal("closed enum JSON accepted unknown number")
	}
	if err := proto.UnmarshalText("closed: 9", new(pb.Sample)); err == nil {
		t.Fatal("closed enum text accepted unknown number")
	}
}

func TestValidationAndGroups(t *testing.T) {
	for _, bad := range []proto.Message{&pb.Sample{Checked: proto.String(string([]byte{255}))}, new(pb.Required)} {
		if _, err := proto.Marshal(bad); err == nil {
			t.Fatal("invalid message encoded")
		}
	}
	if err := proto.Unmarshal(nil, new(pb.Required)); err == nil {
		t.Fatal("required field not checked")
	}
	if err := proto.Unmarshal(wire("2201ff"), new(pb.Sample)); err == nil {
		t.Fatal("UTF8 verification missing")
	}
	if err := proto.Unmarshal(wire("2a01ff3201ff"), new(pb.Sample)); err != nil {
		t.Fatal("NONE validation rejected bytes:", err)
	}
	m := &pb.Sample{Delimited: &pb.Child{Value: proto.Int32(1)}, DelimitedMany: []*pb.Child{{Value: proto.Int32(2)}, {Value: proto.Int32(3)}}, Prefixed: &pb.Child{Value: proto.Int32(4)}}
	b := marshal(t, m)
	if !bytes.Equal(b, wire("6b08016c73080274730803747a020804")) {
		t.Fatalf("message encodings: %x", b)
	}
	decoded := new(pb.Sample)
	if err := proto.Unmarshal(b, decoded); err != nil || !proto.Equal(m, decoded) {
		t.Fatalf("group roundtrip: %+v %v", decoded, err)
	}
	text := proto.MarshalTextString(m)
	back := new(pb.Sample)
	if err := proto.UnmarshalText(text, back); err != nil || !proto.Equal(m, back) {
		t.Fatalf("text groups: %s %v", text, err)
	}
	inherited := &pb.Inherited{Unchecked: proto.String(string([]byte{255})), Delimited: &pb.Child{Value: proto.Int32(1)}, MapPrefixed: map[string]*pb.Child{"k": {Value: proto.Int32(2)}}, Nested: &pb.Inherited_Nested{Unchecked: proto.String(string([]byte{255}))}}
	b = marshal(t, inherited)
	backInherited := new(pb.Inherited)
	if err := proto.Unmarshal(b, backInherited); err != nil || !proto.Equal(inherited, backInherited) {
		t.Fatalf("inherited features: %x %v", b, err)
	}
}

func TestRawDescriptorRetained(t *testing.T) {
	raw, _ := new(pb.Sample).Descriptor()
	z, err := gzip.NewReader(bytes.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	b, err := ioutil.ReadAll(z)
	z.Close()
	if err != nil {
		t.Fatal(err)
	}
	file := new(d.FileDescriptorProto)
	if err := proto.Unmarshal(b, file); err != nil {
		t.Fatal(err)
	}
	if file.GetSyntax() != "editions" || file.GetEdition() != d.Edition_EDITION_2023 {
		t.Fatal("edition rewritten")
	}
	f := file.MessageType[1].Field
	if f[12].GetType() != d.FieldDescriptorProto_TYPE_MESSAGE || f[12].GetOptions().GetFeatures().GetMessageEncoding() != d.FeatureSet_DELIMITED || f[0].Options != nil {
		t.Fatal("lowered descriptor leaked into reflection data")
	}
}

func TestOfficialCompilerOracle(t *testing.T) {
	compiler := os.Getenv("GOGO_EDITION_PROTOC")
	if compiler == "" {
		t.Skip("run compat/check-editions.sh with protoc 36.2 for independent oracle")
	}
	encode := func(name, text string) []byte {
		t.Helper()
		cmd := exec.Command(compiler, "-I../..", "--encode="+name, "compat/editions/edition2023/sample.proto", "compat/editions/edition2023/inherited.proto")
		cmd.Stdin = strings.NewReader(text)
		b, err := cmd.Output()
		if err != nil {
			t.Fatalf("official encode: %v", err)
		}
		return b
	}
	for _, tc := range []struct {
		name, text string
		message    proto.Message
	}{
		{"compat.editions.v2023.Sample", `value: 0 explicit_bytes: "" packed: [0,1,2] expanded:[0,1,2] closed:CLOSED_ONE open:9 delimited:{value:1} delimited_many:[{value:2},{value:3}] prefixed:{value:4} selected:CLOSED_ONE imported:FIRST`, &pb.Sample{Value: proto.Int32(0), ExplicitBytes: []byte{}, Packed: []int32{0, 1, 2}, Expanded: []int32{0, 1, 2}, Closed: pb.Closed_CLOSED_ONE.Enum(), Open: pb.Open(9).Enum(), Delimited: &pb.Child{Value: proto.Int32(1)}, DelimitedMany: []*pb.Child{{Value: proto.Int32(2)}, {Value: proto.Int32(3)}}, Prefixed: &pb.Child{Value: proto.Int32(4)}, Choice: &pb.Sample_Selected{Selected: pb.Closed_CLOSED_ONE}, Imported: pb.Imported_FIRST.Enum()}},
		{"compat.editions.v2023.Inherited", `unchecked:"\377" delimited:{value:1} map_prefixed:{key:"k" value:{value:2}} nested:{unchecked:"\377"}`, &pb.Inherited{Unchecked: proto.String(string([]byte{255})), Delimited: &pb.Child{Value: proto.Int32(1)}, MapPrefixed: map[string]*pb.Child{"k": {Value: proto.Int32(2)}}, Nested: &pb.Inherited_Nested{Unchecked: proto.String(string([]byte{255}))}}},
	} {
		official := encode(tc.name, tc.text)
		ours := marshal(t, tc.message)
		// Protobuf permits different field order (gogo writes oneofs last).
		// Decode our bytes independently, then canonicalize with the oracle.
		cmd := exec.Command(compiler, "-I../..", "--decode="+tc.name, "compat/editions/edition2023/sample.proto", "compat/editions/edition2023/inherited.proto")
		cmd.Stdin = bytes.NewReader(ours)
		text, err := cmd.Output()
		if err != nil {
			t.Fatal(err)
		}
		if canonical := encode(tc.name, string(text)); !bytes.Equal(official, canonical) {
			t.Fatalf("wire meaning differs from official compiler: %s\nofficial %x\ngogo %x", tc.name, official, ours)
		}
		target := reflect.New(reflect.TypeOf(tc.message).Elem()).Interface().(proto.Message)
		if err := proto.Unmarshal(official, target); err != nil || !proto.Equal(tc.message, target) {
			t.Fatalf("official bytes failed decode: %v", err)
		}
	}
	// The final map value decides whether the whole map entry is unknown.
	for _, data := range []string{"5a070a016b10091001", "5a070a016b10011009"} {
		cmd := exec.Command(compiler, "-I../..", "--decode=compat.editions.v2023.Sample", "compat/editions/edition2023/sample.proto")
		cmd.Stdin = bytes.NewReader(wire(data))
		out, err := cmd.Output()
		if err != nil {
			t.Fatal(err)
		}
		ours := new(pb.Sample)
		if err := proto.Unmarshal(wire(data), ours); err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(out), "closed_map") != (len(ours.ClosedMap) > 0) {
			t.Fatalf("map semantics differ: %s %+v", out, ours)
		}
	}
}

func TestEditionExtensions(t *testing.T) {
	unknown := new(pb.Extensible)
	if err := proto.Unmarshal(wire("a00609aa0603000901a206020901"), unknown); err != nil {
		t.Fatal(err)
	}
	if proto.HasExtension(unknown, pb.E_ExtClosed) {
		t.Fatal("unknown closed extension counted as present")
	}
	values, err := proto.GetExtension(unknown, pb.E_ExtPacked)
	if err != nil || !reflect.DeepEqual(values, []pb.Closed{0, 1}) {
		t.Fatalf("closed packed extension: %+v %v", values, err)
	}
	if !bytes.Equal(unknown.XXX_unrecognized, wire("a00609a80609a206020901")) {
		t.Fatalf("extension unknowns not retained: %x", unknown.XXX_unrecognized)
	}
	dst := new(pb.Extensible)
	for _, pair := range []struct {
		desc  *proto.ExtensionDesc
		value interface{}
	}{
		{pb.E_ExtClosed, pb.Closed_CLOSED_ONE.Enum()},
		{pb.E_ExtPacked, []pb.Closed{0, 1}},
		{pb.E_ExtDelimited, &pb.Child{Value: proto.Int32(2)}},
		{pb.E_ExtUnchecked, proto.String(string([]byte{255}))},
		{pb.E_ExtFromForeignScope, proto.String(string([]byte{255}))},
		{pb.E_Declaring_NestedExt, proto.Int32(0)},
	} {
		if err := proto.SetExtension(dst, pair.desc, pair.value); err != nil {
			t.Fatal(err)
		}
	}
	b := marshal(t, dst)
	back := new(pb.Extensible)
	if err := proto.Unmarshal(b, back); err != nil {
		t.Fatal(err)
	}
	for _, desc := range []*proto.ExtensionDesc{pb.E_ExtClosed, pb.E_ExtPacked, pb.E_ExtDelimited, pb.E_ExtUnchecked, pb.E_Declaring_NestedExt, pb.E_ExtFromForeignScope} {
		left, err := proto.GetExtension(dst, desc)
		if err != nil {
			t.Fatal(err)
		}
		right, err := proto.GetExtension(back, desc)
		if err != nil || !sameExtension(left, right) {
			t.Fatalf("extension roundtrip %s: %+v %+v %v", desc.Name, left, right, err)
		}
	}
	compiler := os.Getenv("GOGO_EDITION_PROTOC")
	if compiler != "" {
		cmd := exec.Command(compiler, "-I../..", "--decode=compat.editions.v2023.Extensible", "compat/editions/edition2023/sample.proto", "compat/editions/edition2023/foreign_ext.proto")
		cmd.Stdin = bytes.NewReader(b)
		text, err := cmd.Output()
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(text), "CLOSED_ONE") || !strings.Contains(string(text), "nested_ext") {
			t.Fatalf("official extension decode: %s", text)
		}
		cmd = exec.Command(compiler, "-I../..", "--encode=compat.editions.v2023.Extensible", "compat/editions/edition2023/sample.proto", "compat/editions/edition2023/foreign_ext.proto")
		cmd.Stdin = bytes.NewReader(text)
		official, err := cmd.Output()
		if err != nil {
			t.Fatal(err)
		}
		decoded := new(pb.Extensible)
		if err := proto.Unmarshal(official, decoded); err != nil {
			t.Fatal(err)
		}
		for _, desc := range []*proto.ExtensionDesc{pb.E_ExtClosed, pb.E_ExtPacked, pb.E_ExtDelimited, pb.E_ExtUnchecked, pb.E_Declaring_NestedExt, pb.E_ExtFromForeignScope} {
			a, _ := proto.GetExtension(dst, desc)
			b, err := proto.GetExtension(decoded, desc)
			if err != nil || !sameExtension(a, b) {
				t.Fatalf("official extension roundtrip %s: %v", desc.Name, err)
			}
		}
	}
}

func sameExtension(a, b interface{}) bool {
	if left, ok := a.(proto.Message); ok {
		right, ok := b.(proto.Message)
		return ok && proto.Equal(left, right)
	}
	return reflect.DeepEqual(a, b)
}
