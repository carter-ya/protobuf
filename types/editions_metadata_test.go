package types_test

import (
	"github.com/gogo/protobuf/jsonpb"
	"github.com/gogo/protobuf/jsonpb/standard"
	"github.com/gogo/protobuf/proto"
	"github.com/gogo/protobuf/types"
	"reflect"
	"strings"
	"testing"
)

func TestEditionsMetadataAccessors(t *testing.T) {
	for _, value := range []proto.Message{
		&types.Type{Name: "M", Syntax: types.Syntax_SYNTAX_EDITIONS, Edition: "2026"},
		&types.Enum{Name: "E", Syntax: types.Syntax_SYNTAX_EDITIONS, Edition: "2024"},
		&types.Api{Name: "A", Syntax: types.Syntax_SYNTAX_EDITIONS, Edition: "2023"},
		&types.Method{Name: "Call", Syntax: types.Syntax_SYNTAX_EDITIONS, Edition: "2023"},
	} {
		edition := value.(interface{ GetEdition() string }).GetEdition()
		for _, standardMode := range []bool{false, true} {
			var text string
			var err error
			if standardMode {
				text, err = (&standard.Marshaler{}).MarshalToString(value)
			} else {
				text, err = (&jsonpb.Marshaler{}).MarshalToString(value)
			}
			if err != nil || !strings.Contains(text, `"edition":"`+edition+`"`) || !strings.Contains(text, `"syntax":"SYNTAX_EDITIONS"`) {
				t.Fatalf("%T: %s %v", value, text, err)
			}
			next := reflect.New(reflect.TypeOf(value).Elem()).Interface().(proto.Message)
			if standardMode {
				err = standard.UnmarshalString(text, next)
			} else {
				err = jsonpb.UnmarshalString(text, next)
			}
			if err != nil || !proto.Equal(value, next) {
				t.Fatalf("%T: %v", value, err)
			}
		}
		wire, err := proto.Marshal(value)
		if err != nil {
			t.Fatal(err)
		}
		next := reflect.New(reflect.TypeOf(value).Elem()).Interface().(proto.Message)
		if err := proto.Unmarshal(wire, next); err != nil || !proto.Equal(value, next) {
			t.Fatalf("binary %T: %v", value, err)
		}
	}
	if (*types.Type)(nil).GetEdition() != "" || (*types.Enum)(nil).GetEdition() != "" || (*types.Api)(nil).GetEdition() != "" || (*types.Method)(nil).GetEdition() != "" {
		t.Fatal("nil accessor default")
	}
}
