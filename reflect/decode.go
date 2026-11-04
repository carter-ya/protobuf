package gogoreflect

import (
	"errors"
	"fmt"

	gogo "github.com/gogo/protobuf/proto"
	"google.golang.org/protobuf/encoding/protowire"
	v2 "google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/runtime/protoiface"
)

// V2's reflective decoder treats CLOSED enums as open. Delegate only such
// scalar/map occurrences to the qualified native decoder. Other occurrences
// use the official reflective decoder, including its resolver, unknown-field,
// recursion and merging rules. Processing occurrences in wire order preserves
// oneof replacement and message merge semantics across the two decoders.
var nativeMethods = protoiface.Methods{
	Flags: protoiface.SupportUnmarshalDiscardUnknown,
	Unmarshal: func(in protoiface.UnmarshalInput) (protoiface.UnmarshalOutput, error) {
		out := protoiface.UnmarshalOutput{}
		if in.Depth <= 0 {
			return out, fmt.Errorf("gogoreflect: recursion limit exceeded")
		}
		m := in.Message.(*message)
		provided := in.Resolver
		for {
			parent, ok := provided.(nativeResolver)
			if !ok {
				break
			}
			provided = parent.provided
		}
		resolver := nativeResolver{provided: provided, owner: m}
		options := v2.UnmarshalOptions{Merge: true, AllowPartial: true, DiscardUnknown: in.Flags&protoiface.UnmarshalDiscardUnknown != 0, Resolver: resolver, RecursionLimit: in.Depth}
		slow := &slowView{Message: m}
		defer slow.flushUnknownExtensions()
		var occurrence *message
		for buf := in.Buf; len(buf) > 0; {
			number, wire, taglen := protowire.ConsumeTag(buf)
			if taglen < 0 {
				return out, protowire.ParseError(taglen)
			}
			used := protowire.ConsumeFieldValue(number, wire, buf[taglen:])
			if used < 0 {
				return out, protowire.ParseError(used)
			}
			raw := buf[:taglen+used]
			buf = buf[taglen+used:]
			fd := m.info.desc.Fields().ByNumber(number)
			if fd == nil && m.info.desc.ExtensionRanges().Has(number) {
				if x, err := resolver.FindExtensionByNumber(m.info.desc.FullName(), number); err == nil {
					fd = x.TypeDescriptor()
				} else if err != protoregistry.NotFound {
					return out, err
				}
			}
			enum := fd
			if fd != nil && fd.IsMap() {
				enum = fd.MapValue()
			}
			if enum != nil && enum.Enum() != nil && enum.Enum().IsClosed() {
				if occurrence == nil {
					occurrence = m.info.New().(*message)
				} else {
					occurrence.Reset()
				}
				if err := m.mergeClosedEnumOccurrence(raw, fd, occurrence, options.DiscardUnknown); err != nil {
					return out, err
				}
			} else if err := options.Unmarshal(raw, slow); err != nil {
				return out, err
			}
		}
		return out, nil
	},
}

// Decode into an empty reusable carrier so native required/extension validation
// visits only this occurrence. Merge its accepted values in wire order, leaving
// final required-field completeness to V2's outer CheckInitialized pass. In
// particular, never revalidate or re-encode the accumulated enum list/map/store.
func (m *message) mergeClosedEnumOccurrence(raw []byte, fd protoreflect.FieldDescriptor, occurrence *message, discard bool) error {
	err := gogo.UnmarshalMerge(raw, occurrence.native())
	var required *gogo.RequiredNotSetError
	if err != nil && !errors.As(err, &required) {
		return err
	}
	unknown := occurrence.nativeField(occurrence.info.unknown).Bytes()
	switch {
	case fd.IsList():
		incoming := occurrence.Get(fd).List()
		if incoming.Len() != 0 {
			list := m.Mutable(fd).List()
			for i := 0; i < incoming.Len(); i++ {
				list.Append(incoming.Get(i))
			}
		}
	case fd.IsMap():
		occurrence.Get(fd).Map().Range(func(key protoreflect.MapKey, value protoreflect.Value) bool {
			m.Mutable(fd).Map().Set(key, value)
			return true
		})
	default:
		// A known implicit-presence zero must clear an earlier nonzero value,
		// even though Has reports false for the accepted default value.
		if occurrence.Has(fd) || (!fd.HasPresence() && len(unknown) == 0) {
			m.Set(fd, occurrence.Get(fd))
		}
	}
	if !discard && len(unknown) != 0 {
		field := m.nativeField(m.info.unknown)
		field.SetBytes(append(field.Bytes(), unknown...))
	}
	return nil
}

// slowView is private to one official reflective decode. The official decoder
// appends one validated occurrence to GetUnknown before calling SetUnknown.
// Keep regular wire in its native append buffer and collect extension changes
// until this decode returns, so neither prefix is copied for each occurrence.
type slowView struct {
	protoreflect.Message
	unknownExtensions map[int32][]byte
}

func (s *slowView) ProtoReflect() protoreflect.Message   { return s }
func (s *slowView) Interface() protoreflect.ProtoMessage { return s }
func (s *slowView) ProtoMethods() *protoiface.Methods    { return nil }
func (s *slowView) GetUnknown() protoreflect.RawFields {
	m := s.Message.(*message)
	return m.nativeField(m.info.unknown).Bytes()
}
func (s *slowView) SetUnknown(b protoreflect.RawFields) {
	m := s.Message.(*message)
	m.requireMutable()
	regular := m.nativeField(m.info.unknown)
	if m.info.desc.ExtensionRanges().Len() == 0 {
		regular.SetBytes(b)
		return
	}
	// Only the new, already validated occurrence needs classification.
	raw := b[len(regular.Bytes()):]
	number, _, n := protowire.ConsumeField(raw)
	if n < 0 || n != len(raw) {
		panic("gogoreflect: invalid internal unknown append")
	}
	if !m.info.desc.ExtensionRanges().Has(number) || gogo.RegisteredExtensions(m.native())[int32(number)] != nil {
		regular.SetBytes(b)
		return
	}
	if s.unknownExtensions == nil {
		fields, err := gogo.UnregisteredExtensionWire(m.native())
		if err != nil {
			panic(err)
		}
		if fields == nil {
			fields = make(map[int32][]byte)
		}
		s.unknownExtensions = fields
	}
	s.unknownExtensions[int32(number)] = append(s.unknownExtensions[int32(number)], raw...)
}
func (s *slowView) flushUnknownExtensions() {
	if s.unknownExtensions != nil {
		if err := gogo.ReplaceUnregisteredExtensions(s.Message.(*message).native(), s.unknownExtensions); err != nil {
			panic(err)
		}
	}
}

type extensionResolver interface {
	FindExtensionByName(protoreflect.FullName) (protoreflect.ExtensionType, error)
	FindExtensionByNumber(protoreflect.FullName, protoreflect.FieldNumber) (protoreflect.ExtensionType, error)
}
type nativeResolver struct {
	provided extensionResolver
	owner    *message
}

func (r nativeResolver) FindExtensionByNumber(name protoreflect.FullName, n protoreflect.FieldNumber) (protoreflect.ExtensionType, error) {
	x, err := r.provided.FindExtensionByNumber(name, n)
	if err == nil {
		if name != r.owner.info.desc.FullName() {
			return nil, fmt.Errorf("gogoreflect: foreign extension owner")
		}
		e := gogo.RegisteredExtensions(r.owner.native())[int32(n)]
		if e == nil {
			return nil, fmt.Errorf("gogoreflect: resolver extension %s has no native binding", x.TypeDescriptor().FullName())
		}
		native, er := extensionFor(e)
		if er != nil {
			return nil, er
		}
		fd := x.TypeDescriptor().Descriptor()
		if fd != native.desc {
			return nil, fmt.Errorf("gogoreflect: resolver extension descriptor differs from native binding")
		}
		return x, nil
	}
	if err != protoregistry.NotFound || r.provided != protoregistry.GlobalTypes {
		return nil, err
	}
	if name == r.owner.info.desc.FullName() {
		if e := gogo.RegisteredExtensions(r.owner.native())[int32(n)]; e != nil {
			return extensionFor(e)
		}
	}
	return nil, protoregistry.NotFound
}
func (r nativeResolver) FindExtensionByName(name protoreflect.FullName) (protoreflect.ExtensionType, error) {
	return r.provided.FindExtensionByName(name)
}
