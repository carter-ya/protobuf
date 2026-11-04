package editions_test

import (
	"encoding/json"
	pb "github.com/gogo/protobuf/compat/editions/edition2026"
	"github.com/gogo/protobuf/jsonpb"
	"github.com/gogo/protobuf/proto"
	"testing"
)

func TestEdition2026JSON(t *testing.T) {
	for _, n := range []pb.Status{0, 1, 2, 7} {
		m := &pb.Sample{Status: n.Enum(), Statuses: []pb.Status{n}, Values: map[string]pb.Status{"k": n}}
		j, err := (&jsonpb.Marshaler{}).MarshalToString(m)
		if err != nil {
			t.Fatal(err)
		}
		var object map[string]interface{}
		if err := json.Unmarshal([]byte(j), &object); err != nil {
			t.Fatal(err)
		}
		if n == 1 && object["status"] != "ok\"line\n雪" {
			t.Fatalf("custom name not used: %s", j)
		}
		if n == 2 && object["status"] != "" {
			t.Fatalf("empty name not used: %s", j)
		}
		back := new(pb.Sample)
		if err := jsonpb.UnmarshalString(j, back); err != nil || !proto.Equal(m, back) {
			t.Fatalf("JSON roundtrip: %s %v", j, err)
		}
	}
	for _, name := range []string{"STATUS_OK", "STATUS_ALIAS", "ok\"line\n雪"} {
		value, _ := json.Marshal(name)
		m := new(pb.Sample)
		if err := jsonpb.UnmarshalString(`{"status":`+string(value)+`}`, m); err != nil || m.GetStatus() != pb.Status_STATUS_OK {
			t.Fatalf("JSON alias %q: %v", name, err)
		}
	}
	m := &pb.Sample{Status: pb.Status_STATUS_OK.Enum()}
	j, err := (&jsonpb.Marshaler{EnumsAsInts: true}).MarshalToString(m)
	if err != nil || j != `{"status":1}` {
		t.Fatalf("integer enum output: %s %v", j, err)
	}
	if pb.Status_STATUS_OK.String() != "STATUS_OK" || proto.EnumValueMap("compat.editions.v2026.Status")["STATUS_ALIAS"] != 1 {
		t.Fatal("text names changed")
	}
	text := proto.MarshalTextString(m)
	back := new(pb.Sample)
	if err := proto.UnmarshalText(text, back); err != nil || !proto.Equal(m, back) {
		t.Fatalf("text roundtrip: %s %v", text, err)
	}
}
