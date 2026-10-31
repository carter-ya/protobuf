// Command feature-audit inventories every feature in the pinned official schemas.
// Passing the audit means the inventory is complete, not that listed gaps vanish.
package main

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"

	"github.com/gogo/protobuf/proto"
	d "github.com/gogo/protobuf/protoc-gen-gogo/descriptor"
)

type feature struct {
	Values   map[string]int32 `json:"values,omitempty"`
	Name     string           `json:"name"`
	Number   int32            `json:"number"`
	Status   string           `json:"status"`
	Owner    string           `json:"owner"`
	Evidence []string         `json:"evidence"`
}
type inventory struct {
	Reference string    `json:"reference"`
	Core      []feature `json:"core"`
	Go        []feature `json:"go"`
	Remaining []feature `json:"remaining"`
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
func messages(files []*d.FileDescriptorProto) map[string]*d.DescriptorProto {
	out := map[string]*d.DescriptorProto{}
	var visit func(string, []*d.DescriptorProto)
	visit = func(scope string, ms []*d.DescriptorProto) {
		for _, m := range ms {
			name := scope + "." + m.GetName()
			out[name] = m
			visit(name, m.NestedType)
		}
	}
	for _, f := range files {
		visit("."+f.GetPackage(), f.MessageType)
	}
	return out
}
func enumValues(files []*d.FileDescriptorProto) map[string]map[string]int32 {
	out := map[string]map[string]int32{}
	var visit func(string, []*d.EnumDescriptorProto, []*d.DescriptorProto)
	visit = func(scope string, enums []*d.EnumDescriptorProto, ms []*d.DescriptorProto) {
		for _, e := range enums {
			values := map[string]int32{}
			for _, v := range e.Value {
				values[v.GetName()] = v.GetNumber()
			}
			out[scope+"."+e.GetName()] = values
		}
		for _, m := range ms {
			visit(scope+"."+m.GetName(), m.EnumType, m.NestedType)
		}
	}
	for _, f := range files {
		visit("."+f.GetPackage(), f.EnumType, f.MessageType)
	}
	return out
}
func main() {
	if len(os.Args) != 5 {
		panic("usage: feature-audit official.pb inventory.json repository report.json")
	}
	raw, err := ioutil.ReadFile(os.Args[1])
	must(err)
	var official d.FileDescriptorSet
	must(proto.Unmarshal(raw, &official))
	raw, err = ioutil.ReadFile(os.Args[2])
	must(err)
	var manifest inventory
	must(json.Unmarshal(raw, &manifest))
	compressed, _ := (*d.FeatureSet)(nil).Descriptor()
	z, err := gzip.NewReader(bytes.NewReader(compressed))
	must(err)
	raw, err = ioutil.ReadAll(z)
	must(err)
	must(z.Close())
	var compiled d.FileDescriptorProto
	must(proto.Unmarshal(raw, &compiled))
	upstream := messages(official.File)
	ours := messages([]*d.FileDescriptorProto{&compiled})
	statuses := map[string]int{}
	upstreamEnums := enumValues(official.File)
	compiledEnums := enumValues([]*d.FileDescriptorProto{&compiled})
	audit := func(name string, rows []feature, compiled bool) {
		schema := upstream[name]
		if schema == nil {
			panic("missing official schema: " + name)
		}
		if len(rows) != len(schema.Field) {
			panic("feature inventory changed: " + name)
		}
		seen := map[string]bool{}
		for _, row := range rows {
			if seen[row.Name] {
				panic("duplicate feature: " + row.Name)
			}
			seen[row.Name] = true
			var field *d.FieldDescriptorProto
			for _, f := range schema.Field {
				if f.GetName() == row.Name {
					field = f
				}
			}
			if field == nil || field.GetNumber() != row.Number {
				panic("unreviewed feature: " + row.Name)
			}
			if row.Status != "implemented" && row.Status != "compiler" && row.Status != "gap" {
				panic("invalid status")
			}
			if row.Owner == "" || len(row.Evidence) == 0 {
				panic("feature lacks evidence: " + row.Name)
			}
			for _, path := range row.Evidence {
				_, err := os.Stat(filepath.Join(os.Args[3], path))
				must(err)
			}
			statuses[row.Status]++
			if field.GetType() == d.FieldDescriptorProto_TYPE_ENUM {
				values := upstreamEnums[field.GetTypeName()]
				if !reflect.DeepEqual(row.Values, values) {
					panic("enum value inventory changed: " + row.Name)
				}
				if compiled && !reflect.DeepEqual(compiledEnums[field.GetTypeName()], values) {
					panic("compiled enum values differ: " + row.Name)
				}
			}
			if compiled {
				var found *d.FieldDescriptorProto
				for _, f := range ours[name].Field {
					if f.GetName() == row.Name {
						found = f
					}
				}
				if found == nil || found.GetNumber() != field.GetNumber() || found.GetType() != field.GetType() || found.GetTypeName() != field.GetTypeName() || !proto.Equal(found.Options, field.Options) {
					panic("compiled feature metadata differs: " + row.Name)
				}
			}
		}
	}
	audit(".google.protobuf.FeatureSet", manifest.Core, true)
	audit(".pb.GoFeatures", manifest.Go, false)
	for _, row := range manifest.Remaining {
		if row.Status == "gap" {
			statuses["gap"]++
		}
	}
	report := map[string]interface{}{"reference": manifest.Reference, "core_features": len(manifest.Core), "go_features": len(manifest.Go), "statuses": statuses, "complete_inventory": true, "all_features_implemented": statuses["gap"] == 0, "inventory": manifest}
	raw, err = json.MarshalIndent(report, "", "  ")
	must(err)
	must(ioutil.WriteFile(os.Args[4], append(raw, '\n'), 0644))
	fmt.Printf("Feature inventory complete: %d core, %d Go; %d explicit gaps\n", len(manifest.Core), len(manifest.Go), statuses["gap"])
}
