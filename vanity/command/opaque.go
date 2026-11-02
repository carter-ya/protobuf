package command

import (
	"fmt"
	"strings"

	"github.com/gogo/protobuf/proto"
	"github.com/gogo/protobuf/protoc-gen-gogo/descriptor"
	plugin "github.com/gogo/protobuf/protoc-gen-gogo/plugin"
)

func shiftAnnotations(info *descriptor.GeneratedCodeInfo, delta int32) {
	for _, a := range info.GetAnnotation() {
		if a.Begin != nil {
			a.Begin = proto.Int32(a.GetBegin() + delta)
		}
		if a.End != nil {
			a.End = proto.Int32(a.GetEnd() + delta)
		}
	}
}

func tagOutput(file *plugin.CodeGeneratorResponse_File, expression string) error {
	prefix := "//go:build " + expression + "\n// +build " + expression + "\n\n"
	if strings.HasSuffix(file.GetName(), ".meta") {
		info := new(descriptor.GeneratedCodeInfo)
		if err := proto.UnmarshalText(file.GetContent(), info); err != nil {
			return err
		}
		shiftAnnotations(info, int32(len(prefix)))
		file.Content = proto.String(proto.CompactTextString(info))
	} else {
		file.Content = proto.String(prefix + file.GetContent())
		if file.GeneratedCodeInfo != nil {
			shiftAnnotations(file.GeneratedCodeInfo, int32(len(prefix)))
		}
	}
	return nil
}

// Only files containing HYBRID messages are duplicated. Scoped OPEN and direct
// OPAQUE messages keep their selection when the HYBRID messages switch layouts.
func mergeOpaqueBuilds(normal, opaque *plugin.CodeGeneratorResponse, files []string) error {
	names := make(map[string]string)
	for _, name := range files {
		base := strings.TrimSuffix(name, ".pb.go")
		names[name] = base + "_protoopaque.pb.go"
		names[name+".meta"] = base + "_protoopaque.pb.go.meta"
		names[base+"pb_test.go"] = base + "_protoopaquepb_test.go"
	}
	needed, found := make(map[string]bool), make(map[string]bool)
	for _, file := range normal.File {
		if _, ok := names[file.GetName()]; !ok {
			continue
		}
		needed[file.GetName()] = true
		if err := tagOutput(file, "!protoopaque"); err != nil {
			return fmt.Errorf("HYBRID annotations: %v", err)
		}
	}
	for _, file := range opaque.File {
		name := file.GetName()
		if !needed[name] {
			continue
		}
		if err := tagOutput(file, "protoopaque"); err != nil {
			return fmt.Errorf("OPAQUE annotations: %v", err)
		}
		file.Name = proto.String(names[name])
		normal.File = append(normal.File, file)
		found[name] = true
	}
	for name := range needed {
		if !found[name] {
			return fmt.Errorf("missing OPAQUE alternate output for %s", name)
		}
	}
	return nil
}
