// Package modern experiments with explicit, detached gogo/V2 conversion.
// It does not install ProtoReflect methods or global registrations.
package modern

import (
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"reflect"
	"strings"

	gogo "github.com/gogo/protobuf/proto"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"
)

// ErrLossyConversion means the target Go representation cannot retain the
// modern message's values, presence or unknown fields. Apply leaves it intact.
var ErrLossyConversion = errors.New("modern: conversion loses protobuf data")

// ErrUnsupportedSemantics rejects cases the official dynamic runtime cannot
// faithfully expose through reflection, even when a wire roundtrip succeeds.
var ErrUnsupportedSemantics = errors.New("modern: unsupported reflection semantics")

// Registry owns the descriptors and types for a conversion experiment. Include
// all dependencies and Any payloads in its descriptor set. Finish building it
// before concurrent use; do not mutate Files or Types during conversions.
type Registry struct {
	Files       *protoregistry.Files
	Types       *protoregistry.Types
	sourceFiles map[string]*descriptorpb.FileDescriptorProto
}

// NewRegistry loads a complete protoc --include_imports descriptor set. A second
// pass decodes dynamically declared options, including Edition 2026 JSON names.
// Nothing is registered in the official runtime's process-wide registries.
func NewRegistry(raw []byte) (*Registry, error) {
	set := new(descriptorpb.FileDescriptorSet)
	if err := proto.Unmarshal(raw, set); err != nil {
		return nil, err
	}
	files, err := protodesc.NewFiles(set)
	if err != nil {
		return nil, err
	}
	first := &Registry{Files: files, Types: new(protoregistry.Types)}
	if err := first.registerTypes(); err != nil {
		return nil, err
	}
	set.Reset()
	if err := (proto.UnmarshalOptions{Resolver: first.Types}).Unmarshal(raw, set); err != nil {
		return nil, err
	}
	files, err = protodesc.NewFiles(set)
	if err != nil {
		return nil, err
	}
	result := &Registry{Files: files, Types: new(protoregistry.Types)}
	if err := result.registerTypes(); err != nil {
		return nil, err
	}
	// Decode again with the final extension descriptors so dynamic option
	// messages use the same descriptor identities as source validation.
	set.Reset()
	if err := (proto.UnmarshalOptions{Resolver: result.Types}).Unmarshal(raw, set); err != nil {
		return nil, err
	}
	result.sourceFiles = make(map[string]*descriptorpb.FileDescriptorProto)
	for _, file := range set.File {
		file.SourceCodeInfo = nil // generated runtime descriptors omit locations
		result.sourceFiles[file.GetName()] = file
	}
	return result, nil
}

func (r *Registry) registerTypes() error {
	var add func(protoreflect.MessageDescriptors, protoreflect.ExtensionDescriptors) error
	add = func(messages protoreflect.MessageDescriptors, extensions protoreflect.ExtensionDescriptors) error {
		for i := 0; i < extensions.Len(); i++ {
			if err := r.Types.RegisterExtension(dynamicpb.NewExtensionType(extensions.Get(i))); err != nil {
				return err
			}
		}
		for i := 0; i < messages.Len(); i++ {
			m := messages.Get(i)
			if err := r.Types.RegisterMessage(dynamicpb.NewMessageType(m)); err != nil {
				return err
			}
			if err := add(m.Messages(), m.Extensions()); err != nil {
				return err
			}
		}
		return nil
	}
	var err error
	r.Files.RangeFiles(func(f protoreflect.FileDescriptor) bool {
		err = add(f.Messages(), f.Extensions())
		return err == nil
	})
	return err
}

func validTarget(m gogo.Message) bool {
	v := reflect.ValueOf(m)
	return v.IsValid() && v.Kind() == reflect.Ptr && !v.IsNil() && v.Elem().Kind() == reflect.Struct
}

// runtimeDescriptor recognizes only the historical path aliases used by
// generated gogo dependencies. Missing runtime metadata must not be replaced
// by the caller's potentially incompatible registry descriptor.
func runtimeDescriptor(path string) []byte {
	if raw := gogo.FileDescriptor(path); len(raw) != 0 {
		return raw
	}
	if path == "gogoproto/gogo.proto" || path == "github.com/gogo/protobuf/gogoproto/gogo.proto" {
		return gogo.FileDescriptor("gogo.proto")
	}
	if path == "google/protobuf/descriptor.proto" {
		return gogo.FileDescriptor("descriptor.proto")
	}
	return nil
}

func (r *Registry) decodeSource(raw []byte, path string) (*descriptorpb.FileDescriptorProto, error) {
	z, err := gzip.NewReader(bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	data, err := io.ReadAll(z)
	z.Close()
	if err != nil {
		return nil, err
	}
	file := new(descriptorpb.FileDescriptorProto)
	if err := (proto.UnmarshalOptions{Resolver: r.Types}).Unmarshal(data, file); err != nil {
		return nil, err
	}
	file.SourceCodeInfo = nil
	if path != "" && file.GetName() != path {
		alias := file.GetName() == "gogo.proto" && (path == "gogoproto/gogo.proto" || path == "github.com/gogo/protobuf/gogoproto/gogo.proto") || file.GetName() == "descriptor.proto" && path == "google/protobuf/descriptor.proto"
		if !alias {
			return nil, fmt.Errorf("modern: dependency %q declares %q", path, file.GetName())
		}
		file.Name = proto.String(path)
	}
	return file, nil
}

func (r *Registry) checkSourceFile(file *descriptorpb.FileDescriptorProto, seen map[string]bool) error {
	path := file.GetName()
	if seen[path] {
		return nil
	}
	if !proto.Equal(file, r.sourceFiles[path]) {
		return fmt.Errorf("modern: source descriptor %q does not match registry", path)
	}
	seen[path] = true
	for _, dependency := range file.Dependency {
		raw := runtimeDescriptor(dependency)
		if len(raw) == 0 {
			return fmt.Errorf("modern: runtime descriptor for dependency %q is unavailable", dependency)
		}
		imported, err := r.decodeSource(raw, dependency)
		if err != nil {
			return err
		}
		if err := r.checkSourceFile(imported, seen); err != nil {
			return err
		}
	}
	return nil
}

// sourceName validates the complete ordinary import closure before resolving
// the message path, including types whose file contributes no generated Go
// message (such as an imported enum-only file).
func (r *Registry) sourceName(m gogo.Message) (protoreflect.FullName, error) {
	d, ok := m.(interface{ Descriptor() ([]byte, []int) })
	if !ok {
		return "", fmt.Errorf("modern: %T has no source descriptor", m)
	}
	raw, path := d.Descriptor()
	file, err := r.decodeSource(raw, "")
	if err != nil {
		return "", err
	}
	if err := r.checkSourceFile(file, map[string]bool{}); err != nil {
		return "", err
	}
	if len(path) == 0 {
		return "", fmt.Errorf("modern: missing message descriptor path")
	}
	parts := []string{}
	if file.GetPackage() != "" {
		parts = append(parts, file.GetPackage())
	}
	messages := file.MessageType
	for _, index := range path {
		if index < 0 || index >= len(messages) {
			return "", fmt.Errorf("modern: invalid message descriptor path")
		}
		message := messages[index]
		parts = append(parts, message.GetName())
		messages = message.NestedType
	}
	return protoreflect.FullName(strings.Join(parts, ".")), nil
}

// Snapshot encodes a gogo message into an independent modern dynamic message.
// Mutation on either side never updates the other. Descriptor/type mismatches,
// missing dependencies and missing required fields return errors.
func (r *Registry) Snapshot(m gogo.Message) (*dynamicpb.Message, error) {
	if !validTarget(m) {
		return nil, fmt.Errorf("modern: expected a non-nil gogo message pointer")
	}
	name, err := r.sourceName(m)
	if err != nil {
		return nil, err
	}
	t, err := r.Types.FindMessageByName(name)
	if err != nil {
		return nil, err
	}
	wire, err := gogo.Marshal(m)
	if err != nil {
		return nil, err
	}
	result := dynamicpb.NewMessage(t.Descriptor())
	if err := (proto.UnmarshalOptions{Resolver: r.Types}).Unmarshal(wire, result); err != nil {
		return nil, err
	}
	if err := r.validateMessage(result.ProtoReflect()); err != nil {
		return nil, err
	}
	return result, nil
}

// validateMessage catches semantic mismatches even when wire roundtrips
// succeed, including closed enums nested inside Any payloads. Mutable snapshots
// can also contain cycles, which have no finite protobuf wire representation.
func (r *Registry) validateMessage(root protoreflect.Message) error {
	active := map[protoreflect.Message]bool{}
	var walk func(protoreflect.Message, int) error
	walk = func(m protoreflect.Message, depth int) error {
		if depth > 10000 || active[m] {
			return fmt.Errorf("%w: cyclic or excessively nested message", ErrUnsupportedSemantics)
		}
		active[m] = true
		defer delete(active, m)
		if m.Descriptor().FullName() == "google.protobuf.Any" {
			fields := m.Descriptor().Fields()
			url := m.Get(fields.ByName("type_url")).String()
			data := m.Get(fields.ByName("value")).Bytes()
			if url != "" || len(data) != 0 {
				t, err := r.Types.FindMessageByURL(url)
				if err != nil {
					return fmt.Errorf("modern: Any payload %q: %w", url, err)
				}
				payload := t.New().Interface()
				if err := (proto.UnmarshalOptions{Resolver: r.Types}).Unmarshal(data, payload); err != nil {
					return err
				}
				if err := walk(payload.ProtoReflect(), depth+1); err != nil {
					return err
				}
			}
		}
		var err error
		value := func(fd protoreflect.FieldDescriptor, v protoreflect.Value) {
			if err != nil {
				return
			}
			// The official dynamic decoder currently treats unknown closed
			// enum values as known, contrary to the gogo Editions contract.
			if fd.Kind() == protoreflect.EnumKind && fd.Enum().IsClosed() && fd.Enum().Values().ByNumber(v.Enum()) == nil {
				err = fmt.Errorf("%w: unknown closed enum %s=%d", ErrUnsupportedSemantics, fd.FullName(), v.Enum())
			} else if fd.Kind() == protoreflect.MessageKind || fd.Kind() == protoreflect.GroupKind {
				err = walk(v.Message(), depth+1)
			}
		}
		m.Range(func(fd protoreflect.FieldDescriptor, v protoreflect.Value) bool {
			if fd.IsMap() {
				v.Map().Range(func(_ protoreflect.MapKey, entry protoreflect.Value) bool {
					value(fd.MapValue(), entry)
					return err == nil
				})
			} else if fd.IsList() {
				for i := 0; i < v.List().Len() && err == nil; i++ {
					value(fd, v.List().Get(i))
				}
			} else {
				value(fd, v)
			}
			return err == nil
		})
		return err
	}
	return walk(root, 0)
}

// Apply explicitly replaces dst from a detached snapshot. It validates a fresh
// gogo value by converting back through the same descriptors before committing;
// a representation that drops data is rejected. Callers must synchronize access
// to dst, just as for ordinary protobuf mutation.
func (r *Registry) Apply(dst gogo.Message, src *dynamicpb.Message) error {
	if !validTarget(dst) || src == nil {
		return fmt.Errorf("modern: expected non-nil messages")
	}
	name, err := r.sourceName(dst)
	if err != nil {
		return err
	}
	t, err := r.Types.FindMessageByName(name)
	if err != nil {
		return err
	}
	if src.Descriptor() != t.Descriptor() {
		return fmt.Errorf("modern: snapshot descriptor does not match %T", dst)
	}
	if err := r.validateMessage(src.ProtoReflect()); err != nil {
		return err
	}
	wire, err := proto.Marshal(src)
	if err != nil {
		return err
	}
	candidate := reflect.New(reflect.TypeOf(dst).Elem()).Interface().(gogo.Message)
	if err := gogo.Unmarshal(wire, candidate); err != nil {
		return err
	}
	roundtrip, err := r.Snapshot(candidate)
	if err != nil {
		return err
	}
	if !proto.Equal(src, roundtrip) {
		return ErrLossyConversion
	}
	reflect.ValueOf(dst).Elem().Set(reflect.ValueOf(candidate).Elem())
	return nil
}
