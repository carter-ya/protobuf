package standard

import (
	"bytes"
	"github.com/gogo/protobuf/proto"
	"testing"
)

func TestStandardShortByteEscapes(t *testing.T) {
	for _, tc := range []struct {
		input string
		want  []byte
	}{
		{`\0`, []byte{0}}, {`\00`, []byte{0}}, {`\7tail`, []byte{7, 't', 'a', 'i', 'l'}},
		{`\12`, []byte{10}}, {`\1234`, []byte{'S', '4'}}, {`\18`, []byte{1, '8'}},
		{`\xA`, []byte{10}}, {`\xFHello`, []byte{15, 'H', 'e', 'l', 'l', 'o'}},
		{`\x213`, []byte{'!', '3'}}, {`\x00`, []byte{0}}, {`\303\251`, []byte("é")},
	} {
		for _, field := range []string{"bytes", "checked", "unchecked"} {
			var message probe
			if err := UnmarshalString(field+`: "`+tc.input+`"`, &message); err != nil {
				t.Fatalf("%s: %v", tc.input, err)
			}
			var got []byte
			switch field {
			case "bytes":
				got = message.Bytes
			case "checked":
				got = []byte(message.Checked)
			case "unchecked":
				got = []byte(message.Unchecked)
			}
			if !bytes.Equal(got, tc.want) {
				t.Fatalf("%s decoded %x want %x", tc.input, got, tc.want)
			}
			text, err := (&Marshaler{}).MarshalToString(&message)
			if err != nil {
				t.Fatal(err)
			}
			var back probe
			if err := UnmarshalString(text, &back); err != nil || !proto.Equal(&message, &back) {
				t.Fatalf("escape roundtrip %s: %v", text, err)
			}
		}
	}
}
func TestStandardInvalidByteEscapes(t *testing.T) {
	for _, escape := range []string{`\x`, `\xg`, `\X0f`, `\400`, `\777`, `\u12`, `\U0001000`, `\8`} {
		if err := UnmarshalString(`bytes: "`+escape+`"`, new(probe)); err == nil {
			t.Errorf("accepted invalid escape %s", escape)
		}
	}
	for _, escape := range []string{`\0`, `\12`, `\xA`, `\18`} {
		if err := proto.UnmarshalText(`bytes: "`+escape+`"`, new(probe)); err == nil {
			t.Errorf("legacy short escape acceptance changed: %s", escape)
		}
	}
	var message probe
	if err := proto.UnmarshalText(`bytes: "\X0f"`, &message); err != nil || !bytes.Equal(message.Bytes, []byte{15}) {
		t.Fatalf("legacy uppercase hex changed: %v", err)
	}
}

func TestStandardTextLiteralSourceEncoding(t *testing.T) {
	for _, raw := range []string{string([]byte{248}), string([]byte{255}), string([]byte{0})} {
		for _, field := range []string{"bytes", "checked", "unchecked"} {
			if err := UnmarshalString(field+`: "`+raw+`"`, new(probe)); err == nil {
				t.Fatalf("invalid literal source accepted: %x", raw)
			}
		}
	}
	var message probe
	if err := UnmarshalString(`bytes: "\xff\000"`, &message); err != nil || !bytes.Equal(message.Bytes, []byte{255, 0}) {
		t.Fatalf("escaped bytes changed: %x %v", message.Bytes, err)
	}
	if err := proto.UnmarshalText(`bytes: "`+string([]byte{248})+`"`, new(probe)); err != nil {
		t.Fatalf("legacy raw-byte tolerance changed: %v", err)
	}
}
