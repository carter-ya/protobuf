package generator

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"sort"

	"github.com/gogo/protobuf/proto"
	d "github.com/gogo/protobuf/protoc-gen-gogo/descriptor"
)

func (g *Generator) generateReflectionDependencies(file *FileDescriptor) {
	selected := false
	for _, m := range file.Messages() {
		selected = selected || g.ReflectMessage(m)
	}
	if !selected {
		return
	}
	extras := make(map[string]*d.FileDescriptorProto)
	seen := make(map[string]bool)
	var collect func(string)
	collect = func(name string) {
		if extras[name] != nil {
			return
		}
		f := g.allFilesByName[name]
		if f == nil {
			g.Fail("missing reflection option dependency ", name)
		}
		raw := f.original
		if raw == nil {
			raw = f.FileDescriptorProto
		}
		copy := proto.Clone(raw).(*d.FileDescriptorProto)
		copy.SourceCodeInfo = nil
		extras[name] = copy
		for _, dep := range append(append([]string(nil), raw.Dependency...), raw.OptionDependency...) {
			collect(dep)
		}
	}
	var visit func(*FileDescriptor)
	visit = func(f *FileDescriptor) {
		if seen[f.GetName()] {
			return
		}
		seen[f.GetName()] = true
		raw := f.original
		if raw == nil {
			raw = f.FileDescriptorProto
		}
		for _, dep := range raw.OptionDependency {
			collect(dep)
		}
		for _, dep := range raw.Dependency {
			if next := g.allFilesByName[dep]; next != nil {
				visit(next)
			}
		}
	}
	visit(file)
	if len(extras) == 0 {
		return
	}
	names := make([]string, 0, len(extras))
	for n := range extras {
		names = append(names, n)
	}
	sort.Strings(names)
	set := &d.FileDescriptorSet{}
	for _, n := range names {
		set.File = append(set.File, extras[n])
	}
	raw, err := proto.Marshal(set)
	if err != nil {
		g.Fail(err.Error())
	}
	var buf bytes.Buffer
	z := gzip.NewWriter(&buf)
	z.Write(raw)
	z.Close()
	data := buf.Bytes()
	name := file.VarName() + "_reflection_options"
	runtime := g.reflectionRuntimeImport()
	g.P("func init() { if err := ", runtime, ".RegisterFileSet(", name, "); err != nil { panic(err) } }")
	g.P("var ", name, " = []byte{")
	g.In()
	for len(data) > 0 {
		n := 16
		if len(data) < n {
			n = len(data)
		}
		line := ""
		for _, b := range data[:n] {
			line += fmt.Sprintf("0x%02x,", b)
		}
		g.P(line)
		data = data[n:]
	}
	g.Out()
	g.P("}")
}
