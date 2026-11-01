// Package gogoreflect provides live V2 reflection over native gogo messages.
// It is an optional module; the legacy runtime has no dependency on V2.
package gogoreflect

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"reflect"
	"strings"
	"sync"

	gogo "github.com/gogo/protobuf/proto"
	v2 "google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"
)

type schemaUniverse struct {
	sync.Mutex
	files        *protoregistry.Files
	types        *protoregistry.Types
	raw          map[string][]byte
	supplemental map[string][]byte
}

var schemas = schemaUniverse{files: new(protoregistry.Files), types: new(protoregistry.Types), raw: make(map[string][]byte)}
var models sync.Map // reflect.Type -> *messageType, immutable after publication

func inflate(raw []byte) ([]byte, error) {
	r, err := gzip.NewReader(bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	defer r.Close()
	b, err := io.ReadAll(io.LimitReader(r, (64<<20)+1))
	if len(b) > 64<<20 {
		return nil, fmt.Errorf("gogoreflect: descriptor exceeds 64 MiB")
	}
	return b, err
}
func canonicalPath(path string) string {
	switch path {
	case "gogo.proto", "gogoproto/gogo.proto", "github.com/gogo/protobuf/gogoproto/gogo.proto":
		return "gogoproto/gogo.proto"
	case "descriptor.proto", "google/protobuf/descriptor.proto":
		return "google/protobuf/descriptor.proto"
	}
	return path
}
func (u *schemaUniverse) source(path string) []byte {
	if b := gogo.FileDescriptor(path); len(b) > 0 {
		return b
	}
	if path == "gogoproto/gogo.proto" || path == "github.com/gogo/protobuf/gogoproto/gogo.proto" {
		if b := gogo.FileDescriptor("gogo.proto"); len(b) > 0 {
			return b
		}
	}
	if path == "google/protobuf/descriptor.proto" {
		if b := gogo.FileDescriptor("descriptor.proto"); len(b) > 0 {
			return b
		}
	}
	if b := u.supplemental[canonicalPath(path)]; b != nil {
		return compressMetadata(b)
	}
	return nil
}
func (u *schemaUniverse) load(raw []byte, path string, active map[string]bool) (protoreflect.FileDescriptor, error) {
	b, err := inflate(raw)
	if err != nil {
		return nil, err
	}
	f := new(descriptorpb.FileDescriptorProto)
	if err = v2.Unmarshal(b, f); err != nil {
		return nil, err
	}
	name := canonicalPath(f.GetName())
	if path != "" && canonicalPath(path) != name {
		return nil, fmt.Errorf("gogoreflect: descriptor %q declares %q", path, f.GetName())
	}
	f.Name = v2.String(name)
	for i, dep := range f.Dependency {
		f.Dependency[i] = canonicalPath(dep)
	}
	for i, dep := range f.OptionDependency {
		f.OptionDependency[i] = canonicalPath(dep)
	}
	b, err = (v2.MarshalOptions{Deterministic: true}).Marshal(f)
	if err != nil {
		return nil, err
	}

	if previous, ok := u.raw[name]; ok {
		if !bytes.Equal(previous, b) {
			return nil, fmt.Errorf("gogoreflect: conflicting source descriptor %q", name)
		}
		return u.files.FindFileByPath(name)
	}
	if active[name] {
		return nil, fmt.Errorf("gogoreflect: import cycle at %q", name)
	}
	active[name] = true
	defer delete(active, name)
	for _, dep := range append(append([]string(nil), f.Dependency...), f.OptionDependency...) {
		raw := u.source(dep)
		if len(raw) == 0 {
			return nil, fmt.Errorf("gogoreflect: missing runtime descriptor %q", dep)
		}
		if _, err = u.load(raw, dep, active); err != nil {
			return nil, err
		}
	}
	// Resolve custom options using dependency declarations and declarations in
	// this very file before publishing its canonical descriptor identity.
	first, err := protodesc.NewFile(f, u.files)
	if err != nil {
		return nil, err
	}
	local := new(protoregistry.Types)
	registerExtensions(first, local)
	resolver := optionResolver{local, u.types}
	f.Reset()
	if err = (v2.UnmarshalOptions{Resolver: resolver}).Unmarshal(b, f); err != nil {
		return nil, err
	}
	final, err := protodesc.NewFile(f, u.files)
	if err != nil {
		return nil, err
	}
	if err = u.files.RegisterFile(final); err != nil {
		return nil, err
	}
	registerExtensions(final, u.types)
	u.raw[name] = append([]byte(nil), b...)
	return final, nil
}

type optionResolver struct{ local, shared *protoregistry.Types }

func (r optionResolver) FindExtensionByName(n protoreflect.FullName) (protoreflect.ExtensionType, error) {
	if t, e := r.local.FindExtensionByName(n); e == nil {
		return t, nil
	}
	return r.shared.FindExtensionByName(n)
}
func (r optionResolver) FindExtensionByNumber(n protoreflect.FullName, k protoreflect.FieldNumber) (protoreflect.ExtensionType, error) {
	if t, e := r.local.FindExtensionByNumber(n, k); e == nil {
		return t, nil
	}
	return r.shared.FindExtensionByNumber(n, k)
}
func registerExtensions(f protoreflect.FileDescriptor, types *protoregistry.Types) {
	var add func(protoreflect.MessageDescriptors, protoreflect.ExtensionDescriptors)
	add = func(ms protoreflect.MessageDescriptors, xs protoreflect.ExtensionDescriptors) {
		for i := 0; i < xs.Len(); i++ {
			if err := types.RegisterExtension(dynamicpb.NewExtensionType(xs.Get(i))); err != nil {
				panic(err)
			}
		}
		for i := 0; i < ms.Len(); i++ {
			add(ms.Get(i).Messages(), ms.Get(i).Extensions())
		}
	}
	add(f.Messages(), f.Extensions())
}
func descriptorOf(t reflect.Type) (protoreflect.MessageDescriptor, error) {
	m, ok := reflect.Zero(t).Interface().(interface{ Descriptor() ([]byte, []int) })
	if !ok {
		return nil, fmt.Errorf("gogoreflect: %v has no generated descriptor", t)
	}
	raw, path := m.Descriptor()
	f, err := schemas.load(raw, "", make(map[string]bool))
	if err != nil {
		return nil, err
	}
	ms := f.Messages()
	var md protoreflect.MessageDescriptor
	for _, i := range path {
		if i < 0 || i >= ms.Len() {
			return nil, fmt.Errorf("gogoreflect: invalid message path for %v", t)
		}
		md = ms.Get(i)
		ms = md.Messages()
	}
	if md == nil {
		return nil, fmt.Errorf("gogoreflect: empty message path for %v", t)
	}
	return md, nil
}

type fieldInfo struct {
	fd      protoreflect.FieldDescriptor
	index   int
	wrapper reflect.Type
	typ     reflect.Type
}
type messageType struct {
	typ     reflect.Type
	desc    protoreflect.MessageDescriptor
	fields  map[protoreflect.FieldNumber]*fieldInfo
	unknown int
}

func model(t reflect.Type) (*messageType, error) {
	if m, ok := models.Load(t); ok {
		return m.(*messageType), nil
	}
	schemas.Lock()
	defer schemas.Unlock()
	if m, ok := models.Load(t); ok {
		return m.(*messageType), nil
	}
	pending := make(map[reflect.Type]*messageType)
	var build func(reflect.Type) (*messageType, error)
	build = func(t reflect.Type) (*messageType, error) {
		if m, ok := models.Load(t); ok {
			return m.(*messageType), nil
		}
		if m := pending[t]; m != nil {
			return m, nil
		}
		if t.Kind() != reflect.Ptr || t.Elem().Kind() != reflect.Struct {
			return nil, fmt.Errorf("gogoreflect: %v is not a native message pointer", t)
		}
		md, err := descriptorOf(t)
		if err != nil {
			return nil, err
		}
		if md.Options().(*descriptorpb.MessageOptions).GetMessageSetWireFormat() {
			return nil, fmt.Errorf("gogoreflect: MessageSet is unsupported")
		}
		if md.ExtensionRanges().Len() > 0 {
			if _, ok := reflect.Zero(t).Interface().(interface{ GetExtensions() *[]byte }); ok {
				return nil, fmt.Errorf("gogoreflect: %s requires map-backed extensions", md.FullName())
			}
		}
		m := &messageType{typ: t, desc: md, fields: make(map[protoreflect.FieldNumber]*fieldInfo), unknown: -1}
		pending[t] = m
		st := t.Elem()
		props := gogo.GetProperties(st)
		for i := 0; i < st.NumField(); i++ {
			sf := st.Field(i)
			if sf.Name == "XXX_unrecognized" && sf.Type == reflect.TypeOf([]byte(nil)) {
				m.unknown = i
			}
			p := props.Prop[i]
			if p.Tag <= 0 {
				continue
			}
			fd := md.Fields().ByNumber(protoreflect.FieldNumber(p.Tag))
			if fd == nil {
				return nil, fmt.Errorf("gogoreflect: tag %d absent from %s", p.Tag, md.FullName())
			}
			if p.CustomType != "" || p.StdTime || p.StdDuration || p.WktPointer {
				return nil, fmt.Errorf("gogoreflect: %s uses a custom/std mapping", fd.FullName())
			}
			m.fields[fd.Number()] = &fieldInfo{fd: fd, index: i, typ: sf.Type}
		}
		for _, p := range props.OneofTypes {
			fd := md.Fields().ByNumber(protoreflect.FieldNumber(p.Prop.Tag))
			m.fields[fd.Number()] = &fieldInfo{fd: fd, index: p.Field, wrapper: p.Type, typ: p.Type.Elem().Field(0).Type}
		}
		if m.unknown < 0 {
			return nil, fmt.Errorf("gogoreflect: %s must retain unknown fields", md.FullName())
		}
		for i := 0; i < md.Fields().Len(); i++ {
			fd := md.Fields().Get(i)
			fi := m.fields[fd.Number()]
			if fi == nil {
				return nil, fmt.Errorf("gogoreflect: missing Go field %s", fd.FullName())
			}
			if st.Field(fi.index).PkgPath != "" {
				return nil, fmt.Errorf("gogoreflect: unexported field %s", fd.FullName())
			}
			typ := fi.typ
			target := fd
			if fd.IsMap() {
				if typ.Kind() != reflect.Map {
					return nil, fmt.Errorf("gogoreflect: map representation for %s", fd.FullName())
				}
				if err = checkScalar(fd.MapKey(), typ.Key()); err != nil {
					return nil, err
				}
				typ = typ.Elem()
				target = fd.MapValue()
			} else if fd.IsList() {
				if typ.Kind() != reflect.Slice {
					return nil, fmt.Errorf("gogoreflect: list representation for %s", fd.FullName())
				}
				typ = typ.Elem()
			}
			if target.Message() != nil {
				if typ.Kind() != reflect.Ptr {
					return nil, fmt.Errorf("gogoreflect: %s requires pointer message storage", fd.FullName())
				}
				child, er := build(typ)
				if er != nil {
					return nil, er
				}
				if child.desc != target.Message() {
					return nil, fmt.Errorf("gogoreflect: descriptor mismatch for %s", fd.FullName())
				}
			} else {
				if !fd.IsMap() && !fd.IsList() && fi.wrapper == nil && fd.HasPresence() && fd.Kind() != protoreflect.BytesKind {
					if typ.Kind() != reflect.Ptr {
						return nil, fmt.Errorf("gogoreflect: %s loses presence", fd.FullName())
					}
					typ = typ.Elem()
				}
				if er := checkScalar(target, typ); er != nil {
					return nil, er
				}
			}
			if target.Enum() != nil && target.Enum().IsClosed() {
				tag := st.Field(fi.index).Tag.Get("protobuf")
				if fd.IsMap() {
					tag = st.Field(fi.index).Tag.Get("protobuf_val")
				}
				if fi.wrapper != nil {
					tag = fi.wrapper.Elem().Field(0).Tag.Get("protobuf")
				}
				if !strings.Contains(tag, "enum_closed=") {
					return nil, fmt.Errorf("gogoreflect: %s requires enforce_closed_enums regeneration", fd.FullName())
				}
			}
		}
		return m, nil
	}
	m, err := build(t)
	if err != nil {
		return nil, err
	}
	for t, m := range pending {
		models.Store(t, m)
	}
	return m, nil
}
func checkScalar(fd protoreflect.FieldDescriptor, t reflect.Type) error {
	kinds := map[protoreflect.Kind]reflect.Kind{protoreflect.BoolKind: reflect.Bool, protoreflect.EnumKind: reflect.Int32, protoreflect.Int32Kind: reflect.Int32, protoreflect.Sint32Kind: reflect.Int32, protoreflect.Sfixed32Kind: reflect.Int32, protoreflect.Int64Kind: reflect.Int64, protoreflect.Sint64Kind: reflect.Int64, protoreflect.Sfixed64Kind: reflect.Int64, protoreflect.Uint32Kind: reflect.Uint32, protoreflect.Fixed32Kind: reflect.Uint32, protoreflect.Uint64Kind: reflect.Uint64, protoreflect.Fixed64Kind: reflect.Uint64, protoreflect.FloatKind: reflect.Float32, protoreflect.DoubleKind: reflect.Float64, protoreflect.StringKind: reflect.String, protoreflect.BytesKind: reflect.Slice}
	if t.Kind() != kinds[fd.Kind()] || (fd.Kind() == protoreflect.BytesKind && t.Elem().Kind() != reflect.Uint8) {
		return fmt.Errorf("gogoreflect: incompatible %v storage for %s", t, fd.FullName())
	}
	return nil
}
