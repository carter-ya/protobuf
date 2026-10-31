// Go support for Protocol Buffers - Google's data interchange format
//
// Copyright 2010 The Go Authors.  All rights reserved.
// https://github.com/golang/protobuf
//
// Redistribution and use in source and binary forms, with or without
// modification, are permitted provided that the following conditions are
// met:
//
//     * Redistributions of source code must retain the above copyright
// notice, this list of conditions and the following disclaimer.
//     * Redistributions in binary form must reproduce the above
// copyright notice, this list of conditions and the following disclaimer
// in the documentation and/or other materials provided with the
// distribution.
//     * Neither the name of Google Inc. nor the names of its
// contributors may be used to endorse or promote products derived from
// this software without specific prior written permission.
//
// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS
// "AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT
// LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR
// A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT
// OWNER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL,
// SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT
// LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE,
// DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY
// THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
// (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
// OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.

package proto

/*
 * Support for message sets.
 */

import (
	"errors"
)

// errNoMessageTypeID occurs when a protocol buffer does not have a message type ID.
// A message type ID is required for storing a protocol buffer in a message set.
var errNoMessageTypeID = errors.New("proto does not have a message type ID")

// The first two types (_MessageSet_Item and messageSet)
// model what the protocol compiler produces for the following protocol message:
//   message MessageSet {
//     repeated group Item = 1 {
//       required int32 type_id = 2;
//       required string message = 3;
//     };
//   }
// That is the MessageSet wire format. We can't use a proto to generate these
// because that would introduce a circular dependency between it and this package.

type _MessageSet_Item struct {
	TypeId  *int32 `protobuf:"varint,2,req,name=type_id"`
	Message []byte `protobuf:"bytes,3,req,name=message"`
}

type messageSet struct {
	Item             []*_MessageSet_Item `protobuf:"group,1,rep"`
	XXX_unrecognized []byte
	// TODO: caching?
}

// Make sure messageSet is a Message.
var _ Message = (*messageSet)(nil)

// messageTypeIder is an interface satisfied by a protocol buffer type
// that may be stored in a MessageSet.
type messageTypeIder interface {
	MessageTypeId() int32
}

func (ms *messageSet) find(pb Message) *_MessageSet_Item {
	mti, ok := pb.(messageTypeIder)
	if !ok {
		return nil
	}
	id := mti.MessageTypeId()
	for _, item := range ms.Item {
		if *item.TypeId == id {
			return item
		}
	}
	return nil
}

func (ms *messageSet) Has(pb Message) bool {
	return ms.find(pb) != nil
}

func (ms *messageSet) Unmarshal(pb Message) error {
	if item := ms.find(pb); item != nil {
		return Unmarshal(item.Message, pb)
	}
	if _, ok := pb.(messageTypeIder); !ok {
		return errNoMessageTypeID
	}
	return nil // TODO: return error instead?
}

func (ms *messageSet) Marshal(pb Message) error {
	msg, err := Marshal(pb)
	if err != nil {
		return err
	}
	if item := ms.find(pb); item != nil {
		// reuse existing item
		item.Message = msg
		return nil
	}

	mti, ok := pb.(messageTypeIder)
	if !ok {
		return errNoMessageTypeID
	}

	mtid := mti.MessageTypeId()
	ms.Item = append(ms.Item, &_MessageSet_Item{
		TypeId:  &mtid,
		Message: msg,
	})
	return nil
}

func (ms *messageSet) Reset()         { *ms = messageSet{} }
func (ms *messageSet) String() string { return CompactTextString(ms) }
func (*messageSet) ProtoMessage()     {}

// Support for the message_set_wire_format message option.

func skipVarint(buf []byte) []byte {
	i := 0
	for ; buf[i]&0x80 != 0; i++ {
	}
	return buf[i+1:]
}

// unmarshalMessageSet decodes the extension map encoded in buf in the message set wire format.
// It is called by Unmarshal methods on protocol buffer messages with the message_set_wire_format option.
func unmarshalMessageSet(buf []byte, exts interface{}) error {
	var err error
	buf, err = normalizeMessageSetInput(buf)
	if err != nil {
		return err
	}
	var m map[int32]Extension
	switch exts := exts.(type) {
	case *XXX_InternalExtensions:
		m = exts.extensionsWrite()
	case map[int32]Extension:
		m = exts
	default:
		return errors.New("proto: not an extension map")
	}

	ms := new(messageSet)
	if err := Unmarshal(buf, ms); err != nil {
		return err
	}
	for _, item := range ms.Item {
		id := *item.TypeId
		msg := item.Message

		// Restore wire type and field number varint, plus length varint.
		// Be careful to preserve duplicate items.
		b := EncodeVarint(uint64(id)<<3 | WireBytes)
		if ext, ok := m[id]; ok {
			var old []byte
			if ext.value != nil {
				// GetExtension discards enc, and SetExtension starts decoded.
				// Materialize that message before concatenating incoming fields.
				value, ok := ext.value.(Message)
				if !ok {
					return errors.New("proto: invalid decoded MessageSet extension")
				}
				old, err = Marshal(value)
				if err != nil {
					return err
				}
			} else {
				// This header is the internal extension representation, not
				// an ordinary wire field. MessageSet type IDs extend past the
				// usual 29-bit field-number limit.
				key, header, err := editionVarint(ext.enc)
				if err != nil {
					return err
				}
				if key != uint64(id)<<3|WireBytes {
					return errors.New("proto: invalid encoded MessageSet extension")
				}
				length, prefix, err := editionVarint(ext.enc[header:])
				if err != nil {
					return err
				}
				if length != uint64(len(ext.enc)-header-prefix) {
					return errors.New("proto: invalid encoded MessageSet length")
				}
				old = ext.enc[header+prefix:]
			}
			joined := make([]byte, 0, len(old)+len(msg))
			joined = append(joined, old...)
			msg = append(joined, msg...)
		}
		b = append(b, EncodeVarint(uint64(len(msg)))...)
		b = append(b, msg...)

		m[id] = Extension{enc: b}
	}
	return nil
}

// A writer unaware of message_set_wire_format may emit ordinary length-delimited
// extension fields. Rewrite those to items before parsing, in place in the field
// sequence, so mixed encodings preserve merge order. Pure group input is reused.
func normalizeMessageSetInput(buf []byte) ([]byte, error) {
	var normalized []byte
	for offset := 0; offset < len(buf); {
		field := buf[offset:]
		end, header, _, err := editionField(field)
		if err != nil {
			return nil, err
		}
		key, _ := DecodeVarint(field)
		if key&7 == WireBytes {
			if normalized == nil {
				normalized = make([]byte, 0, len(buf))
				normalized = append(normalized, buf[:offset]...)
			}
			normalized = append(normalized, 11, 16) // item group, type_id
			normalized = append(normalized, EncodeVarint(key>>3)...)
			normalized = append(normalized, 26) // message, with original length/body
			normalized = append(normalized, field[header:end]...)
			normalized = append(normalized, 12) // end item
		} else if normalized != nil {
			normalized = append(normalized, field[:end]...)
		}
		offset += end
	}
	if normalized == nil {
		return buf, nil
	}
	return normalized, nil
}
