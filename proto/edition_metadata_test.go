package proto_test

import (
	"bytes"
	"reflect"
	"strings"
	"testing"

	"github.com/gogo/protobuf/jsonpb"
	"github.com/gogo/protobuf/proto"
)

// Handwritten tags isolate the runtime contract from generator implementation.
type editionMetadata struct {
	Explicit         []byte           `protobuf:"bytes,1,opt,name=explicit,proto3,explicit"`
	Implicit         []byte           `protobuf:"bytes,2,opt,name=implicit,proto3,utf8_none"`
	Checked          *string          `protobuf:"bytes,3,opt,name=checked,proto3,explicit"`
	Unchecked        string           `protobuf:"bytes,4,opt,name=unchecked,proto3,utf8_none"`
	Enum             *int32           `protobuf:"varint,5,opt,name=enum,enum_closed=1:2"`
	Enums            []int32          `protobuf:"varint,6,rep,packed,name=enums,enum_closed=1:2"`
	Map              map[string]int32 `protobuf:"bytes,7,rep,name=map" protobuf_key:"bytes,1,opt,name=key" protobuf_val:"varint,2,opt,name=value,enum_closed=1:2,def=1"`
	XXX_unrecognized []byte
}

func (*editionMetadata) ProtoMessage()    {}
func (m *editionMetadata) Reset()         { *m = editionMetadata{} }
func (m *editionMetadata) String() string { return proto.CompactTextString(m) }

func TestEditionRuntimePresenceAndUTF8(t *testing.T) {
	m := &editionMetadata{Explicit: []byte{}, Implicit: []byte{}}
	b, err := proto.Marshal(m)
	if err != nil || !bytes.Equal(b, []byte{10, 0}) {
		t.Fatalf("presence: %x %v", b, err)
	}
	if proto.Equal(m, &editionMetadata{}) {
		t.Fatal("explicit empty bytes lost presence")
	}
	if proto.Clone(m).(*editionMetadata).Explicit == nil {
		t.Fatal("clone lost explicit presence")
	}
	dst := &editionMetadata{Explicit: []byte{1}}
	proto.Merge(dst, m)
	if dst.Explicit == nil || len(dst.Explicit) != 0 {
		t.Fatal("merge lost explicit empty bytes")
	}
	if _, err := proto.Marshal(&editionMetadata{Checked: proto.String(string([]byte{255}))}); err == nil {
		t.Fatal("explicit string skipped UTF-8 validation")
	}
	if err := proto.Unmarshal([]byte{26, 1, 255}, new(editionMetadata)); err == nil {
		t.Fatal("explicit string decode skipped UTF-8 validation")
	}
	if _, err := proto.Marshal(&editionMetadata{Unchecked: string([]byte{255})}); err != nil {
		t.Fatal(err)
	}
	if err := proto.Unmarshal([]byte{34, 1, 255}, new(editionMetadata)); err != nil {
		t.Fatal(err)
	}
	j, err := (&jsonpb.Marshaler{EmitDefaults: true}).MarshalToString(new(editionMetadata))
	if err != nil || strings.Contains(j, `"explicit"`) || strings.Contains(j, `"checked"`) {
		t.Fatalf("JSON absent presence: %s %v", j, err)
	}
}

func TestEditionClosedEnumUnknownRetention(t *testing.T) {
	for _, tc := range []struct {
		name           string
		input, unknown []byte
		enums          []int32
		wantMap        map[string]int32
	}{
		{"singular", []byte{40, 9}, []byte{40, 9}, nil, nil},
		{"expanded", []byte{48, 1, 48, 9, 48, 2}, []byte{48, 9}, []int32{1, 2}, nil},
		{"packed", []byte{50, 4, 1, 9, 2, 8}, []byte{48, 9, 48, 8}, []int32{1, 2}, nil},
		{"map unknown", []byte{58, 5, 10, 1, 'k', 16, 9}, []byte{58, 5, 10, 1, 'k', 16, 9}, nil, nil},
		{"map default", []byte{58, 3, 10, 1, 'k'}, nil, nil, map[string]int32{"k": 1}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := new(editionMetadata)
			if err := proto.Unmarshal(tc.input, m); err != nil {
				t.Fatal(err)
			}
			if m.Enum != nil || !reflect.DeepEqual(m.Enums, tc.enums) || !reflect.DeepEqual(m.Map, tc.wantMap) || !bytes.Equal(m.XXX_unrecognized, tc.unknown) {
				t.Fatalf("wrong decode: %+v", m)
			}
			b, err := proto.Marshal(m)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.HasSuffix(b, tc.unknown) {
				t.Fatalf("unknown lost: %x", b)
			}
		})
	}
}

func TestEditionMetadataPropertiesRoundTrip(t *testing.T) {
	var p, q proto.Properties
	p.Parse("varint,5,opt,name=enum,proto3,explicit,utf8_none,enum_closed=-1:1:2")
	q.Parse(p.String())
	if !reflect.DeepEqual(p, q) {
		t.Fatalf("metadata roundtrip: %+v %+v", p, q)
	}
}
