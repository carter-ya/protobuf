// Package standard provides explicit standard Protobuf text entrypoints while
// retaining the historical proto text defaults. See README.md for coverage.
package standard

import (
	"bytes"
	"io"
	"io/ioutil"

	"github.com/gogo/protobuf/proto"
)

// Marshaler configures the standard text encoder. Use keyed fields.
// Unknown fields are omitted unless PrintUnknownFields is set.
type Marshaler struct {
	_                  struct{} // keep future configuration additions compatible with keyed literals
	Compact            bool
	ExpandAny          bool
	PrintUnknownFields bool
}

func (m *Marshaler) Marshal(out io.Writer, message proto.Message) error {
	if m == nil {
		m = new(Marshaler)
	}
	return proto.MarshalTextStandard(&proto.TextMarshaler{Compact: m.Compact, ExpandAny: m.ExpandAny}, out, message, m.PrintUnknownFields)
}
func (m *Marshaler) MarshalToString(message proto.Message) (string, error) {
	var buffer bytes.Buffer
	if err := m.Marshal(&buffer, message); err != nil {
		return "", err
	}
	return buffer.String(), nil
}
func Unmarshal(in io.Reader, message proto.Message) error {
	data, err := ioutil.ReadAll(in)
	if err != nil {
		return err
	}
	return UnmarshalString(string(data), message)
}
func UnmarshalString(input string, message proto.Message) error {
	return proto.UnmarshalTextStandard(input, message)
}
