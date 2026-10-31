package standard

import (
	"strings"
	"testing"

	pb "github.com/gogo/protobuf/jsonpb/jsonpb_test_proto"
	"github.com/gogo/protobuf/proto"
	"github.com/gogo/protobuf/types"
)

func TestStandardExpandedAnyURL(t *testing.T) {
	name := proto.MessageName(new(pb.Widget))
	for _, prefix := range []string{"type.googleapis.com/", "non.default.domain/sub-path_0/~!$&()*+,;=/", "non.default.domain/%2F/"} {
		url := prefix + name
		var value types.Any
		if err := UnmarshalString("["+url+"] {}", &value); err != nil || value.TypeUrl != url {
			t.Fatalf("parse %q: %v", url, err)
		}
		text, err := (&Marshaler{ExpandAny: true}).MarshalToString(&value)
		if err != nil || !strings.Contains(text, "["+url+"]") {
			t.Fatalf("expanded %q: %q %v", url, text, err)
		}
		var back types.Any
		if err := UnmarshalString(text, &back); err != nil || !proto.Equal(&value, &back) {
			t.Fatalf("roundtrip %q: %v", url, err)
		}
	}
	var value types.Any
	if err := UnmarshalString("[type.go # comment\n ogleapis.com/"+name+"] {}", &value); err != nil || value.TypeUrl != "type.googleapis.com/"+name {
		t.Fatalf("URL whitespace/comments: %v", err)
	}
	for _, prefix := range []string{"/", "/domain/", "domain/%ZZ/", "domain/%2/", "domain/%/", "domain/:/", "domain/☃/"} {
		if err := UnmarshalString("["+prefix+name+"] {}", new(types.Any)); err == nil {
			t.Errorf("accepted %q", prefix)
		}
	}
	if err := UnmarshalString("[type.googleapis.com/"+name+"] {}", new(probe)); err == nil {
		t.Fatal("expanded Any in non-Any message")
	}
	// URL mode must end at ']' so punctuation does not become ordinary tokens.
	if err := UnmarshalString("[type.googleapis.com/"+name+"] {} !", new(types.Any)); err == nil {
		t.Fatal("URL lexer mode escaped brackets")
	}
	if err := proto.UnmarshalText("[domain/~!$/"+name+"] {}", new(types.Any)); err == nil {
		t.Fatal("legacy URL grammar changed")
	}
	for _, url := range []string{"domain/%ZZ/" + name, name, "domain/missing.Message"} {
		value := &types.Any{TypeUrl: url}
		text, err := (&Marshaler{ExpandAny: true}).MarshalToString(value)
		if err != nil || !strings.Contains(text, "type_url:") || strings.Contains(text, "[") {
			t.Fatalf("invalid expansion fallback: %q %v", text, err)
		}
		var back types.Any
		if err := UnmarshalString(text, &back); err != nil || !proto.Equal(value, &back) {
			t.Fatalf("fallback roundtrip %q: %v", text, err)
		}
	}
}
