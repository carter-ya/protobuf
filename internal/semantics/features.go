// Package semantics resolves the core Protobuf feature set. It is a generator
// dependency only; the runtime consumes the independent fieldmeta model.
package semantics

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io/ioutil"
	"reflect"
	"sync"

	"github.com/gogo/protobuf/internal/fieldmeta"
	"github.com/gogo/protobuf/proto"
	d "github.com/gogo/protobuf/protoc-gen-gogo/descriptor"
)

type Resolver struct {
	edition  d.Edition
	defaults fieldmeta.Features
	fixed    *d.FeatureSet
}

var tables struct {
	sync.Once
	defaults d.FeatureSetDefaults
	fields   map[string]*d.FieldDescriptorProto
	err      error
}

func load() {
	if err := proto.Unmarshal(compiledDefaults, &tables.defaults); err != nil {
		tables.err = err
		return
	}
	b, indices := (*d.FeatureSet)(nil).Descriptor()
	z, err := gzip.NewReader(bytes.NewReader(b))
	if err != nil {
		tables.err = err
		return
	}
	raw, err := ioutil.ReadAll(z)
	z.Close()
	if err != nil {
		tables.err = err
		return
	}
	var file d.FileDescriptorProto
	if err := proto.Unmarshal(raw, &file); err != nil {
		tables.err = err
		return
	}
	m := file.MessageType[indices[0]]
	for _, i := range indices[1:] {
		m = m.NestedType[i]
	}
	tables.fields = make(map[string]*d.FieldDescriptorProto)
	for _, f := range m.Field {
		tables.fields[f.GetName()] = f
	}
}

func New(edition d.Edition) (*Resolver, error) {
	tables.Do(load)
	if tables.err != nil {
		return nil, tables.err
	}
	if edition < tables.defaults.GetMinimumEdition() || edition > tables.defaults.GetMaximumEdition() {
		return nil, fmt.Errorf("unsupported edition %d", edition)
	}
	var selected *d.FeatureSetDefaults_FeatureSetEditionDefault
	for _, candidate := range tables.defaults.Defaults {
		if candidate.GetEdition() <= edition {
			selected = candidate
		}
	}
	if selected == nil {
		return nil, fmt.Errorf("no feature defaults for edition %d", edition)
	}
	merged := proto.Clone(selected.GetFixedFeatures()).(*d.FeatureSet)
	proto.Merge(merged, selected.GetOverridableFeatures())
	r := &Resolver{edition: edition, fixed: proto.Clone(selected.GetFixedFeatures()).(*d.FeatureSet)}
	r.defaults = fromProto(merged)
	return r, nil
}

func (r *Resolver) Defaults() fieldmeta.Features { return r.defaults }

type featureSpec struct {
	goName, protoName string
	destination       *int32
	allowed           []int32
}

// Resolve returns an independent value. Neither raw descriptors nor inherited
// state are modified. Targets and fixed-feature restrictions come from the
// pinned protocol schema/defaults, rather than file syntax heuristics.
func (r *Resolver) Resolve(parent fieldmeta.Features, local *d.FeatureSet, target d.FieldOptions_OptionTargetType) (fieldmeta.Features, error) {
	result := parent
	if local == nil {
		return result, nil
	}
	specs := []featureSpec{
		{"FieldPresence", "field_presence", &result.FieldPresence, []int32{1, 2, 3}},
		{"EnumType", "enum_type", &result.EnumType, []int32{1, 2}},
		{"RepeatedFieldEncoding", "repeated_field_encoding", &result.RepeatedEncoding, []int32{1, 2}},
		{"Utf8Validation", "utf8_validation", &result.UTF8Validation, []int32{2, 3}},
		{"MessageEncoding", "message_encoding", &result.MessageEncoding, []int32{1, 2}},
		{"JsonFormat", "json_format", &result.JSONFormat, []int32{1, 2}},
		{"EnforceNamingStyle", "enforce_naming_style", &result.NamingStyle, []int32{1, 2, 3}},
		{"DefaultSymbolVisibility", "default_symbol_visibility", &result.SymbolVisibility, []int32{1, 2, 3, 4}},
		{"EnforceProtoLimits", "enforce_proto_limits", &result.ProtoLimits, []int32{1, 2}},
	}
	value, fixed := reflect.ValueOf(local).Elem(), reflect.ValueOf(r.fixed).Elem()
	for _, spec := range specs {
		field := value.FieldByName(spec.goName)
		if field.IsNil() {
			continue
		}
		if !fixed.FieldByName(spec.goName).IsNil() {
			return parent, fmt.Errorf("feature %s cannot be overridden in edition %d", spec.protoName, r.edition)
		}
		info := tables.fields[spec.protoName]
		allowedTarget := false
		for _, t := range info.GetOptions().GetTargets() {
			if t == target {
				allowedTarget = true
			}
		}
		if !allowedTarget {
			return parent, fmt.Errorf("feature %s is not allowed on %s", spec.protoName, target)
		}
		n := int32(field.Elem().Int())
		allowedValue := false
		for _, v := range spec.allowed {
			if v == n {
				allowedValue = true
			}
		}
		if !allowedValue {
			return parent, fmt.Errorf("unknown value %d for feature %s", n, spec.protoName)
		}
		*spec.destination = n
	}
	return result, nil
}

func fromProto(f *d.FeatureSet) fieldmeta.Features {
	return fieldmeta.Features{FieldPresence: int32(f.GetFieldPresence()), EnumType: int32(f.GetEnumType()), RepeatedEncoding: int32(f.GetRepeatedFieldEncoding()), UTF8Validation: int32(f.GetUtf8Validation()), MessageEncoding: int32(f.GetMessageEncoding()), JSONFormat: int32(f.GetJsonFormat()), NamingStyle: int32(f.GetEnforceNamingStyle()), SymbolVisibility: int32(f.GetDefaultSymbolVisibility()), ProtoLimits: int32(f.GetEnforceProtoLimits())}
}

// ForField accounts for structural presence and legacy syntax spellings after
// feature inheritance. A repeated field never gains scalar presence.
func ForField(inherited fieldmeta.Features, f *d.FieldDescriptorProto) fieldmeta.Features {
	out := inherited
	if f.IsRepeated() {
		out.FieldPresence = 0
	} else if f.IsRequired() {
		out.FieldPresence = 3
	} else if f.IsMessage() || f.IsRealOneof() || f.GetProto3Optional() || f.GetExtendee() != "" {
		out.FieldPresence = 1
	}
	if f.Options != nil && f.Options.Packed != nil {
		if f.GetOptions().GetPacked() {
			out.RepeatedEncoding = 1
		} else {
			out.RepeatedEncoding = 2
		}
	}
	if f.GetType() == d.FieldDescriptorProto_TYPE_GROUP {
		out.MessageEncoding = 2
	}
	return out
}
