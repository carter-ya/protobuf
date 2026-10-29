// Package standard provides an explicit Protobuf JSON entrypoint while jsonpb
// retains its historical defaults. See README.md for the supported contracts
// and remaining conformance limitations.
package standard

import (
	"bytes"
	"github.com/gogo/protobuf/jsonpb"
	"github.com/gogo/protobuf/proto"
	"io"
	"strings"
)

// Marshaler has the same configuration fields as jsonpb.Marshaler. Standard
// mode preserves presence and rejects invalid UTF-8. Nil message elements encode as empty messages.
// Custom JSONPB hooks receive the legacy configuration and must themselves
// produce standard JSON.
type Marshaler jsonpb.Marshaler

func (m *Marshaler) Marshal(out io.Writer, message proto.Message) error {
	return jsonpb.MarshalStandard((*jsonpb.Marshaler)(m), out, message)
}
func (m *Marshaler) MarshalToString(message proto.Message) (string, error) {
	var out bytes.Buffer
	if err := m.Marshal(&out, message); err != nil {
		return "", err
	}
	return out.String(), nil
}

// Unmarshaler consumes one whole document, resets the destination, and rejects
// duplicate fields, multiple oneof members, invalid Unicode and null elements.
type Unmarshaler jsonpb.Unmarshaler

func (u *Unmarshaler) Unmarshal(in io.Reader, message proto.Message) error {
	return jsonpb.UnmarshalStandard((*jsonpb.Unmarshaler)(u), in, message)
}
func Unmarshal(in io.Reader, message proto.Message) error {
	return new(Unmarshaler).Unmarshal(in, message)
}
func UnmarshalString(input string, message proto.Message) error {
	return Unmarshal(strings.NewReader(input), message)
}
