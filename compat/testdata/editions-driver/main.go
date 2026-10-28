// Test-only capability bridge. Production generators do not advertise Editions
// until the backend's independent compiler/runtime matrix has passed.
package main

import (
	"github.com/gogo/protobuf/proto"
	d "github.com/gogo/protobuf/protoc-gen-gogo/descriptor"
	p "github.com/gogo/protobuf/protoc-gen-gogo/plugin"
	"github.com/gogo/protobuf/vanity/command"
)

func main() {
	r := command.Generate(command.Read())
	r.SupportedFeatures = proto.Uint64(uint64(p.CodeGeneratorResponse_FEATURE_PROTO3_OPTIONAL | p.CodeGeneratorResponse_FEATURE_SUPPORTS_EDITIONS))
	r.MinimumEdition = proto.Int32(int32(d.Edition_EDITION_2023))
	r.MaximumEdition = proto.Int32(int32(d.Edition_EDITION_2023))
	command.Write(r)
}
