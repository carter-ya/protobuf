package gogoreflect

import (
	"fmt"
	"reflect"
	"strings"

	gogo "github.com/gogo/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
)

// Registry is an explicit V2 resolver for qualified native gogo types. It does
// not modify V2 global registries. Create it after native registration, then
// share it for reads. Root types and their reachable message types are included.
// Pass payload types explicitly when resolving Any. Native extension bindings
// are consulted live, so serial late registration is visible without rebuilding
// message caches; synchronize registration as required by the gogo runtime.
type Registry struct {
	types map[protoreflect.FullName]*messageType
}

func NewRegistry(roots ...gogo.Message) (*Registry, error) {
	r := &Registry{types: make(map[protoreflect.FullName]*messageType)}
	var add func(reflect.Type) error
	add = func(t reflect.Type) error {
		m, err := model(t)
		if err != nil {
			return err
		}
		if previous := r.types[m.desc.FullName()]; previous != nil {
			if previous.typ != m.typ {
				return fmt.Errorf("gogoreflect: multiple Go bindings for %s", m.desc.FullName())
			}
			return nil
		}
		r.types[m.desc.FullName()] = m
		for _, f := range m.fields {
			target := f.fd
			typ := f.typ
			if target.IsMap() {
				target = target.MapValue()
				typ = typ.Elem()
			} else if target.IsList() {
				typ = typ.Elem()
			}
			if target.Message() != nil {
				if err = add(typ); err != nil {
					return err
				}
			}
		}
		for _, e := range gogo.RegisteredExtensions(reflect.Zero(t).Interface().(gogo.Message)) {
			x, err := extensionFor(e)
			if err != nil {
				return err
			}
			if x.desc.Message() != nil {
				typ := x.typ
				if x.desc.IsList() {
					typ = typ.Elem()
				}
				if err = add(typ); err != nil {
					return err
				}
			}
		}
		return nil
	}
	for _, root := range roots {
		if root == nil {
			return nil, fmt.Errorf("gogoreflect: nil registry root")
		}
		if err := add(reflect.TypeOf(root)); err != nil {
			return nil, err
		}
	}
	return r, nil
}
func (r *Registry) FindMessageByName(n protoreflect.FullName) (protoreflect.MessageType, error) {
	if t := r.types[n]; t != nil {
		return t, nil
	}
	return nil, protoregistry.NotFound
}
func (r *Registry) FindMessageByURL(url string) (protoreflect.MessageType, error) {
	i := strings.LastIndex(url, "/")
	return r.FindMessageByName(protoreflect.FullName(url[i+1:]))
}
func (r *Registry) FindExtensionByNumber(name protoreflect.FullName, n protoreflect.FieldNumber) (protoreflect.ExtensionType, error) {
	m := r.types[name]
	if m == nil {
		if descriptorOption(name) {
			schemas.Lock()
			defer schemas.Unlock()
			return schemas.types.FindExtensionByNumber(name, n)
		}
		return nil, protoregistry.NotFound
	}
	e := gogo.RegisteredExtensions(reflect.Zero(m.typ).Interface().(gogo.Message))[int32(n)]
	if e == nil {
		return nil, protoregistry.NotFound
	}
	return extensionFor(e)
}
func (r *Registry) FindExtensionByName(name protoreflect.FullName) (protoreflect.ExtensionType, error) {
	for _, m := range r.types {
		for _, e := range gogo.RegisteredExtensions(reflect.Zero(m.typ).Interface().(gogo.Message)) {
			if protoreflect.FullName(strings.TrimPrefix(e.Name, ".")) == name {
				return extensionFor(e)
			}
		}
	}
	schemas.Lock()
	defer schemas.Unlock()
	if x, err := schemas.types.FindExtensionByName(name); err == nil && descriptorOption(x.TypeDescriptor().ContainingMessage().FullName()) {
		return x, nil
	}
	return nil, protoregistry.NotFound
}
func (r *Registry) RangeExtensionsByMessage(name protoreflect.FullName, fn func(protoreflect.ExtensionType) bool) {
	m := r.types[name]
	if m == nil {
		return
	}
	for _, e := range gogo.RegisteredExtensions(reflect.Zero(m.typ).Interface().(gogo.Message)) {
		x, err := extensionFor(e)
		if err != nil {
			panic(err)
		}
		if !fn(x) {
			return
		}
	}
}

func qualify(info *messageType, seen map[reflect.Type]bool) error {
	if seen[info.typ] {
		return nil
	}
	seen[info.typ] = true
	for _, f := range info.fields {
		fd, typ := f.fd, f.typ
		if fd.IsMap() {
			fd = fd.MapValue()
			typ = typ.Elem()
		} else if fd.IsList() {
			typ = typ.Elem()
		}
		if fd.Message() != nil {
			child, err := model(typ)
			if err != nil {
				return err
			}
			if err = qualify(child, seen); err != nil {
				return err
			}
		}
	}
	for _, e := range gogo.RegisteredExtensions(reflect.Zero(info.typ).Interface().(gogo.Message)) {
		x, err := extensionFor(e)
		if err != nil {
			return err
		}
		if x.desc.Message() != nil {
			typ := x.typ
			if x.desc.IsList() {
				typ = typ.Elem()
			}
			child, err := model(typ)
			if err != nil {
				return err
			}
			if err = qualify(child, seen); err != nil {
				return err
			}
		}
	}
	return nil
}

func descriptorOption(name protoreflect.FullName) bool {
	return strings.HasPrefix(string(name), "google.protobuf.") && (strings.HasSuffix(string(name), "Options") || name == "google.protobuf.FeatureSet")
}
