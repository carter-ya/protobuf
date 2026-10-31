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
	RecursionLimit     int // zero uses the official Go default of 10000
}

func (m *Marshaler) Marshal(out io.Writer, message proto.Message) error {
	if m == nil {
		m = new(Marshaler)
	}
	return proto.MarshalTextStandardWithLimit(&proto.TextMarshaler{Compact: m.Compact, ExpandAny: m.ExpandAny}, out, message, m.PrintUnknownFields, m.RecursionLimit)
}
func (m *Marshaler) MarshalToString(message proto.Message) (string, error) {
	var buffer bytes.Buffer
	if err := m.Marshal(&buffer, message); err != nil {
		return "", err
	}
	return buffer.String(), nil
}
func Unmarshal(in io.Reader, message proto.Message) error {
	return new(Unmarshaler).Unmarshal(in, message)
}

// Unmarshaler configures the standard text decoder. Use keyed fields.
type Unmarshaler struct {
	_              struct{}
	RecursionLimit int // zero uses the official Go default of 10000
}

func (u *Unmarshaler) Unmarshal(in io.Reader, message proto.Message) error {
	data, err := ioutil.ReadAll(in)
	if err != nil {
		return err
	}
	return u.UnmarshalString(string(data), message)
}
func UnmarshalString(input string, message proto.Message) error {
	return new(Unmarshaler).UnmarshalString(input, message)
}

func (u *Unmarshaler) UnmarshalString(input string, message proto.Message) error {
	if u == nil {
		u = new(Unmarshaler)
	}
	return proto.UnmarshalTextStandardWithLimit(input, message, u.RecursionLimit)
}
