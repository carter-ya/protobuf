package generator

import (
	"fmt"
	"github.com/gogo/protobuf/gogoproto"
	"github.com/gogo/protobuf/proto"
	d "github.com/gogo/protobuf/protoc-gen-gogo/descriptor"
)

func optimizeFeature(features *d.FeatureSet, inherited int32) (int32, error) {
	if features == nil || !proto.HasExtension(features, goFeatureExtension) {
		return inherited, nil
	}
	settings, err := readGoFeatures(features)
	if err != nil {
		return 0, err
	}
	if settings.OptimizeMode != nil {
		return int32(*settings.OptimizeMode), nil
	}
	return inherited, nil
}

func prepareOptimizedMessage(file *d.FileDescriptorProto, message *d.DescriptorProto, mode int32) error {
	if mode == 0 {
		return nil
	}
	if gogoproto.IsProtoSizer(file, message) {
		return fmt.Errorf("%s: %s: optimize_mode requires Size rather than protosizer", file.GetName(), message.GetName())
	}
	codecs := []struct{ message, file *proto.ExtensionDesc }{
		{gogoproto.E_Marshaler, gogoproto.E_MarshalerAll},
		{gogoproto.E_Unmarshaler, gogoproto.E_UnmarshalerAll},
		{gogoproto.E_Sizer, gogoproto.E_SizerAll},
	}
	for _, codec := range codecs {
		if !proto.GetBoolExtension(message.Options, codec.message, proto.GetBoolExtension(file.Options, codec.file, true)) {
			return fmt.Errorf("%s: %s: optimize_mode conflicts with disabled codec %s", file.GetName(), message.GetName(), codec.message.Name)
		}
	}
	if message.Options == nil {
		message.Options = &d.MessageOptions{}
	}
	for _, codec := range codecs {
		if err := proto.SetExtension(message.Options, codec.message, proto.Bool(true)); err != nil {
			return err
		}
	}
	return nil
}

func (g *Generator) OptimizeMode(message *Descriptor) int32 {
	if message.File().edition == nil {
		return 0
	}
	return message.File().edition.optimize[message.DescriptorProto]
}

// Selected SPEED parents can call compact children through the same methods.
// When an unselected dependency has no fast methods, use tables for that parent
// instead of generating references to non-existent dependency methods.
func (g *Generator) UseTableCodec(message *Descriptor) bool {
	// Legacy fast plugins do not emit group codecs. The explicit modern profile
	// can retain the native representation using its qualified table wrappers.
	if g.ReflectMessage(message) {
		for _, f := range message.Field {
			if f.GetType() == d.FieldDescriptorProto_TYPE_GROUP {
				return true
			}
		}
	}

	mode := g.OptimizeMode(message)
	if mode == 0 {
		return false
	}
	if mode == 2 {
		return true
	}
	for _, field := range message.Field {
		target := field
		if g.IsMap(field) {
			entry := g.ObjectNamed(field.GetTypeName()).(*Descriptor)
			target = entry.Field[1]
		}
		if !target.IsMessage() && target.GetType() != d.FieldDescriptorProto_TYPE_GROUP {
			continue
		}
		dependency := g.requiredDependency(target)
		df, dm := dependency.File().FileDescriptorProto, dependency.DescriptorProto
		if !gogoproto.IsMarshaler(df, dm) || !gogoproto.IsSizer(df, dm) || !gogoproto.IsUnmarshaler(df, dm) {
			return true
		}
	}
	return false
}
