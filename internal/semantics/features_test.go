package semantics

import (
	"bytes"
	"github.com/gogo/protobuf/proto"
	d "github.com/gogo/protobuf/protoc-gen-gogo/descriptor"
	"strings"
	"testing"
)

func TestCompiledDefaults(t *testing.T) {
	for _, test := range []struct {
		edition                                                  d.Edition
		presence, enum, packed, utf8, naming, visibility, limits int32
	}{
		{d.Edition_EDITION_PROTO2, 1, 2, 2, 3, 2, 1, 1},
		{d.Edition_EDITION_PROTO3, 2, 1, 1, 2, 2, 1, 1},
		{d.Edition_EDITION_2023, 1, 1, 1, 2, 2, 1, 1},
		{d.Edition_EDITION_2024, 1, 1, 1, 2, 1, 2, 1},
		{d.Edition_EDITION_2026, 1, 1, 1, 2, 3, 4, 2},
	} {
		r, err := New(test.edition)
		if err != nil {
			t.Fatal(err)
		}
		f := r.Defaults()
		if f.FieldPresence != test.presence || f.EnumType != test.enum || f.RepeatedEncoding != test.packed || f.UTF8Validation != test.utf8 || f.NamingStyle != test.naming || f.SymbolVisibility != test.visibility || f.ProtoLimits != test.limits {
			t.Fatalf("%s: wrong defaults %+v", test.edition, f)
		}
	}
	if _, err := New(d.Edition(1003)); err == nil {
		t.Fatal("future edition accepted")
	}
}

func TestInheritanceAndOriginalDescriptors(t *testing.T) {
	f := &d.FieldDescriptorProto{Name: proto.String("value"), Type: d.FieldDescriptorProto_TYPE_INT32.Enum(), Label: d.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Options: &d.FieldOptions{Features: &d.FeatureSet{FieldPresence: d.FeatureSet_EXPLICIT.Enum()}}}
	ext := &d.FieldDescriptorProto{Name: proto.String("ext"), Extendee: proto.String(".Other"), Type: d.FieldDescriptorProto_TYPE_INT32.Enum(), Label: d.FieldDescriptorProto_LABEL_OPTIONAL.Enum()}
	file := &d.FileDescriptorProto{Name: proto.String("test.proto"), Syntax: proto.String("editions"), Edition: d.Edition_EDITION_2023.Enum(), Options: &d.FileOptions{Features: &d.FeatureSet{FieldPresence: d.FeatureSet_IMPLICIT.Enum(), Utf8Validation: d.FeatureSet_NONE.Enum()}}, MessageType: []*d.DescriptorProto{{Name: proto.String("Scope"), Options: &d.MessageOptions{Features: &d.FeatureSet{JsonFormat: d.FeatureSet_LEGACY_BEST_EFFORT.Enum()}}, Field: []*d.FieldDescriptorProto{f}, Extension: []*d.FieldDescriptorProto{ext}}}}
	before, err := proto.Marshal(file)
	if err != nil {
		t.Fatal(err)
	}
	resolved, err := ResolveFile(file)
	if err != nil {
		t.Fatal(err)
	}
	if !resolved.Fields[f].HasPresence() || resolved.Fields[f].VerifyUTF8() || resolved.Fields[f].JSONFormat != 2 {
		t.Fatalf("field overrides did not inherit independently: %+v", resolved.Fields[f])
	}
	if !resolved.Fields[ext].HasPresence() || resolved.Fields[ext].JSONFormat != 2 {
		t.Fatal("extension did not inherit its declaration scope")
	}
	after, err := proto.Marshal(file)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) {
		t.Fatal("raw source features were rewritten")
	}
}

func TestInvalidOverrides(t *testing.T) {
	r, err := New(d.Edition_EDITION_2023)
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		features *d.FeatureSet
		target   d.FieldOptions_OptionTargetType
		message  string
	}{
		{&d.FeatureSet{EnforceNamingStyle: d.FeatureSet_STYLE_LEGACY.Enum()}, d.FieldOptions_TARGET_TYPE_FILE, "cannot be overridden"},
		{&d.FeatureSet{FieldPresence: d.FeatureSet_IMPLICIT.Enum()}, d.FieldOptions_TARGET_TYPE_MESSAGE, "not allowed"},
		{&d.FeatureSet{Utf8Validation: d.FeatureSet_Utf8Validation(1).Enum()}, d.FieldOptions_TARGET_TYPE_FIELD, "unknown value"},
	} {
		if _, err := r.Resolve(r.Defaults(), test.features, test.target); err == nil || !strings.Contains(err.Error(), test.message) {
			t.Fatalf("expected %q, got %v", test.message, err)
		}
	}
}

func TestStructuralPresence(t *testing.T) {
	r, _ := New(d.Edition_EDITION_PROTO3)
	base := r.Defaults()
	for _, test := range []struct {
		f    *d.FieldDescriptorProto
		want bool
	}{
		{&d.FieldDescriptorProto{Type: d.FieldDescriptorProto_TYPE_INT32.Enum(), Proto3Optional: proto.Bool(true)}, true},
		{&d.FieldDescriptorProto{Type: d.FieldDescriptorProto_TYPE_MESSAGE.Enum()}, true},
		{&d.FieldDescriptorProto{Type: d.FieldDescriptorProto_TYPE_INT32.Enum(), OneofIndex: proto.Int32(0)}, true},
		{&d.FieldDescriptorProto{Type: d.FieldDescriptorProto_TYPE_MESSAGE.Enum(), Label: d.FieldDescriptorProto_LABEL_REPEATED.Enum()}, false},
	} {
		if got := ForField(base, test.f).HasPresence(); got != test.want {
			t.Errorf("presence got %v want %v", got, test.want)
		}
	}
}
