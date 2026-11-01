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
		slow := slowView{m}
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
				var previous []byte
				if options.DiscardUnknown {
					previous = append([]byte(nil), m.GetUnknown()...)
				}
				if fd.IsExtension() {
					x := m.extension(fd)
					if err := gogo.MaterializeExtensionWire(m.native(), x.native); err != nil {
						return out, err
					}
				}
				err := gogo.UnmarshalMerge(raw, m.native())
				var required *gogo.RequiredNotSetError
				if errors.As(err, &required) {
					err = nil
				}
				if options.DiscardUnknown {
					m.SetUnknown(previous)
				}
				if err != nil {
					return out, err
				}
			} else if err := options.Unmarshal(raw, slow); err != nil {
				return out, err
			}
		}
		return out, nil
	},
}

type slowView struct{ protoreflect.Message }

func (s slowView) ProtoReflect() protoreflect.Message   { return s }
func (s slowView) Interface() protoreflect.ProtoMessage { return s }
func (s slowView) ProtoMethods() *protoiface.Methods    { return nil }

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
