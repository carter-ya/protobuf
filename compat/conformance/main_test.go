package main

import (
	"bytes"
	"encoding/binary"
	pb "example.com/gogo-conformance/pb/protocol"
	"github.com/gogo/protobuf/proto"
	"testing"
)

func TestDispatch(t *testing.T) {
	for _, name := range []string{"protobuf_test_messages.proto2.TestAllTypesProto2", "protobuf_test_messages.proto3.TestAllTypesProto3", "protobuf_test_messages.editions.proto2.TestAllTypesProto2", "protobuf_test_messages.editions.proto3.TestAllTypesProto3", "protobuf_test_messages.editions.TestAllTypesEdition2023"} {
		for _, format := range []pb.WireFormat{pb.WireFormat_PROTOBUF, pb.WireFormat_JSON, pb.WireFormat_TEXT_FORMAT} {
			r := handle(&pb.ConformanceRequest{MessageType: name, Payload: &pb.ConformanceRequest_ProtobufPayload{}, RequestedOutputFormat: format})
			switch r.Result.(type) {
			case *pb.ConformanceResponse_ProtobufPayload, *pb.ConformanceResponse_JsonPayload, *pb.ConformanceResponse_TextPayload:
			default:
				t.Fatalf("%s %v: %v", name, format, r)
			}
		}
	}
}
func TestFrames(t *testing.T) {
	req := &pb.ConformanceRequest{MessageType: "conformance.FailureSet"}
	wire, _ := proto.Marshal(req)
	var input bytes.Buffer
	for i := 0; i < 2; i++ {
		binary.Write(&input, binary.LittleEndian, uint32(len(wire)))
		input.Write(wire)
	}
	var output bytes.Buffer
	if err := serve(&input, &output); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		var n uint32
		if err := binary.Read(&output, binary.LittleEndian, &n); err != nil {
			t.Fatal(err)
		}
		data := make([]byte, n)
		output.Read(data)
		r := new(pb.ConformanceResponse)
		if err := proto.Unmarshal(data, r); err != nil {
			t.Fatal(err)
		}
		if _, ok := r.Result.(*pb.ConformanceResponse_ProtobufPayload); !ok {
			t.Fatal(r)
		}
	}
	if output.Len() != 0 {
		t.Fatal("unexpected output")
	}
	if err := serve(bytes.NewReader([]byte{1, 0}), &output); err == nil {
		t.Fatal("truncated frame accepted")
	}
}
