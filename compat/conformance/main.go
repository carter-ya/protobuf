// The testee speaks the upstream conformance subprocess protocol. Its schemas
// are generated from the pinned upstream distribution, without source edits.
package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"strings"

	e23 "example.com/gogo-conformance/pb/edition2023"
	e2 "example.com/gogo-conformance/pb/editionproto2"
	e3 "example.com/gogo-conformance/pb/editionproto3"
	p2 "example.com/gogo-conformance/pb/proto2"
	p3 "example.com/gogo-conformance/pb/proto3"
	pb "example.com/gogo-conformance/pb/protocol"
	"github.com/gogo/protobuf/jsonpb"
	"github.com/gogo/protobuf/jsonpb/standard"
	"github.com/gogo/protobuf/proto"
	textstandard "github.com/gogo/protobuf/textpb/standard"
)

var standardMode bool

func message(name string) proto.Message {
	switch name {
	case "protobuf_test_messages.proto2.TestAllTypesProto2":
		return new(p2.TestAllTypesProto2)
	case "protobuf_test_messages.proto3.TestAllTypesProto3":
		return new(p3.TestAllTypesProto3)
	case "protobuf_test_messages.editions.proto2.TestAllTypesProto2":
		return new(e2.TestAllTypesProto2)
	case "protobuf_test_messages.editions.proto3.TestAllTypesProto3":
		return new(e3.TestAllTypesProto3)
	case "protobuf_test_messages.editions.TestAllTypesEdition2023":
		return new(e23.TestAllTypesEdition2023)
	}
	return nil
}

func handle(req *pb.ConformanceRequest) (res *pb.ConformanceResponse) {
	defer func() {
		if p := recover(); p != nil {
			res = &pb.ConformanceResponse{Result: &pb.ConformanceResponse_RuntimeError{RuntimeError: fmt.Sprint(p)}}
		}
	}()
	if req.MessageType == "conformance.FailureSet" {
		return &pb.ConformanceResponse{Result: &pb.ConformanceResponse_ProtobufPayload{ProtobufPayload: []byte{}}}
	}
	msg := message(req.MessageType)
	if msg == nil {
		return &pb.ConformanceResponse{Result: &pb.ConformanceResponse_Skipped{Skipped: "unrecognized message type: " + req.MessageType}}
	}
	var err error
	switch p := req.Payload.(type) {
	case *pb.ConformanceRequest_ProtobufPayload:
		err = proto.Unmarshal(p.ProtobufPayload, msg)
	case *pb.ConformanceRequest_JsonPayload:
		u := jsonpb.Unmarshaler{AllowUnknownFields: req.TestCategory == pb.TestCategory_JSON_IGNORE_UNKNOWN_PARSING_TEST}
		if standardMode {
			options := standard.Unmarshaler(u)
			err = options.Unmarshal(strings.NewReader(p.JsonPayload), msg)
		} else {
			err = u.Unmarshal(strings.NewReader(p.JsonPayload), msg)
		}
	case *pb.ConformanceRequest_TextPayload:
		if standardMode {
			err = textstandard.UnmarshalString(p.TextPayload, msg)
		} else {
			err = proto.UnmarshalText(p.TextPayload, msg)
		}
	default:
		return &pb.ConformanceResponse{Result: &pb.ConformanceResponse_Skipped{Skipped: "unsupported input format"}}
	}
	if err != nil {
		return &pb.ConformanceResponse{Result: &pb.ConformanceResponse_ParseError{ParseError: err.Error()}}
	}
	switch req.RequestedOutputFormat {
	case pb.WireFormat_PROTOBUF:
		var wire []byte
		wire, err = proto.Marshal(msg)
		if err == nil {
			return &pb.ConformanceResponse{Result: &pb.ConformanceResponse_ProtobufPayload{ProtobufPayload: wire}}
		}
	case pb.WireFormat_JSON:
		var text string
		if standardMode {
			text, err = (&standard.Marshaler{}).MarshalToString(msg)
		} else {
			text, err = (&jsonpb.Marshaler{}).MarshalToString(msg)
		}
		if err == nil {
			return &pb.ConformanceResponse{Result: &pb.ConformanceResponse_JsonPayload{JsonPayload: text}}
		}
	case pb.WireFormat_TEXT_FORMAT:
		if standardMode {
			var text string
			text, err = (&textstandard.Marshaler{PrintUnknownFields: req.PrintUnknownFields}).MarshalToString(msg)
			if err == nil {
				return &pb.ConformanceResponse{Result: &pb.ConformanceResponse_TextPayload{TextPayload: text}}
			}
		} else {
			return &pb.ConformanceResponse{Result: &pb.ConformanceResponse_TextPayload{TextPayload: proto.MarshalTextString(msg)}}
		}
	default:
		return &pb.ConformanceResponse{Result: &pb.ConformanceResponse_Skipped{Skipped: "unsupported output format"}}
	}
	return &pb.ConformanceResponse{Result: &pb.ConformanceResponse_SerializeError{SerializeError: err.Error()}}
}

func serve(in io.Reader, out io.Writer) error {
	for {
		var size [4]byte
		if _, err := io.ReadFull(in, size[:]); err == io.EOF {
			return nil
		} else if err != nil {
			return err
		}
		n := binary.LittleEndian.Uint32(size[:])
		if n > 64<<20 {
			return fmt.Errorf("conformance frame exceeds 64 MiB")
		}
		data := make([]byte, int(n))
		if _, err := io.ReadFull(in, data); err != nil {
			return err
		}
		req := new(pb.ConformanceRequest)
		if err := proto.Unmarshal(data, req); err != nil {
			return err
		}
		wire, err := proto.Marshal(handle(req))
		if err != nil {
			return err
		}
		binary.LittleEndian.PutUint32(size[:], uint32(len(wire)))
		if _, err = out.Write(size[:]); err != nil {
			return err
		}
		if _, err = out.Write(wire); err != nil {
			return err
		}
	}
}
func main() {
	standardMode = len(os.Args) == 2 && os.Args[1] == "standard"
	if err := serve(os.Stdin, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
