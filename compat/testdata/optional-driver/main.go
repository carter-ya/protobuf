// This test-only driver exposes unfinished optional code generation to the
// compatibility suite. Production entry points advertise no new capability
// until their full entry-point matrix is validated in the next stack layer.
package main

import (
	"github.com/gogo/protobuf/proto"
	"github.com/gogo/protobuf/vanity"
	"github.com/gogo/protobuf/vanity/command"
	"os"
)

func main() {
	req := command.Read()
	if os.Getenv("GOGO_TEST_FAST") == "1" {
		files := vanity.FilterFiles(req.GetProtoFile(), vanity.NotGoogleProtobufDescriptorProto)
		vanity.ForEachFile(files, vanity.TurnOnMarshalerAll)
		vanity.ForEachFile(files, vanity.TurnOnSizerAll)
		vanity.ForEachFile(files, vanity.TurnOnUnmarshalerAll)
	}
	resp := command.Generate(req)
	resp.SupportedFeatures = proto.Uint64(1)
	command.Write(resp)
}
