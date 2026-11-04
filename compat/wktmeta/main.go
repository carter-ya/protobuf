// Command wktmeta verifies that the standard-schema update contains only the
// reviewed Editions metadata additions and Method syntax deprecation.
package main

import (
	"fmt"
	"github.com/gogo/protobuf/proto"
	d "github.com/gogo/protobuf/protoc-gen-gogo/descriptor"
	"io/ioutil"
	"os"
)

func load(path string) (*d.FileDescriptorSet, error) {
	raw, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	set := new(d.FileDescriptorSet)
	err = proto.Unmarshal(raw, set)
	return set, err
}
func check(old, current *d.FileDescriptorSet) error {
	additions := map[string]int32{"google.protobuf.Type": 7, "google.protobuf.Enum": 6, "google.protobuf.Api": 8, "google.protobuf.Method": 8}
	found := make(map[string]bool)
	for _, file := range current.File {
		for _, message := range file.MessageType {
			name := file.GetPackage() + "." + message.GetName()
			if number, ok := additions[name]; ok {
				fields := message.Field[:0]
				for _, field := range message.Field {
					if field.GetName() == "edition" {
						if field.GetNumber() != number || field.GetType() != d.FieldDescriptorProto_TYPE_STRING || field.GetLabel() != d.FieldDescriptorProto_LABEL_OPTIONAL {
							return fmt.Errorf("unexpected edition declaration: %s", name)
						}
						if (name == "google.protobuf.Method") != field.GetOptions().GetDeprecated() {
							return fmt.Errorf("wrong edition deprecation: %s", name)
						}
						if found[name] {
							return fmt.Errorf("duplicate edition field: %s", name)
						}
						found[name] = true
						continue
					}
					if name == "google.protobuf.Method" && field.GetName() == "syntax" {
						if !field.GetOptions().GetDeprecated() {
							return fmt.Errorf("Method.syntax must be deprecated")
						}
						field.Options.Deprecated = nil
						if proto.Size(field.Options) == 0 {
							field.Options = nil
						}
					}
					fields = append(fields, field)
				}
				message.Field = fields
			}
		}
		for _, enum := range file.EnumType {
			if file.GetPackage()+"."+enum.GetName() != "google.protobuf.Syntax" {
				continue
			}
			values := enum.Value[:0]
			for _, value := range enum.Value {
				if value.GetName() == "SYNTAX_EDITIONS" {
					if value.GetNumber() != 2 || found["Syntax"] {
						return fmt.Errorf("invalid Editions syntax value")
					}
					found["Syntax"] = true
					continue
				}
				values = append(values, value)
			}
			enum.Value = values
		}
	}
	if len(found) != 5 {
		return fmt.Errorf("missing additions: %v", found)
	}
	if !proto.Equal(old, current) {
		return fmt.Errorf("existing schema declarations changed outside the reviewed metadata update")
	}
	return nil
}
func main() {
	if len(os.Args) != 3 {
		fmt.Fprintln(os.Stderr, "usage: wktmeta old.pb current.pb")
		os.Exit(2)
	}
	old, err := load(os.Args[1])
	if err == nil {
		var current *d.FileDescriptorSet
		current, err = load(os.Args[2])
		if err == nil {
			err = check(old, current)
		}
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println("Existing fields, defaults and options preserved; four edition fields and syntax value verified.")
}
