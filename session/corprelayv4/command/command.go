// Package command defines the various corp-relay-v4@google.com commands.
package command

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

// Tag represents the tag for an in-band command, see
// https://chromium.googlesource.com/apps/libapps/+/HEAD/nassh/docs/relay-protocol.md#commands.
type Tag uint16

func (t Tag) String() string {
	switch t {
	case TagConnectSuccess:
		return "CONNECT_SUCCESS"
	case TagReconnectSuccess:
		return "RECONNECT_SUCCESS"
	case TagData:
		return "DATA"
	case TagAck:
		return "ACK"
	default:
		return "unknown"
	}
}

const (
	// TagConnectSuccess is the first command after a /v4/connect request.
	TagConnectSuccess Tag = 1

	// TagReconnectSuccess is the first command after a /v4/reconnect request.
	TagReconnectSuccess Tag = 2

	// TagData signals new data for the client.
	TagData Tag = 4

	// TagAck represents a server acking client data.
	TagAck Tag = 7
)

const (
	// MaxArrayLen is the maximum allowed length for an array in a relay command.
	MaxArrayLen = 16 * 1024

	// TagLen is the length of a Tag.
	TagLen = 2

	// AckLen is the length of an ack in a relay command.
	AckLen = 8

	// SIDLen is the length of the sid_length field in a CONNECT_SUCCESS command.
	SIDLen = 4

	// MaxSIDLen is the maximum allowed length for a SID.
	// NOTE: We use ASCII-encoded RFC 4122 UUIDs, though we allow shorter values for compatibility.
	MaxSIDLen = 36

	// DataLen is the length of the data_length field in a DATA command.
	DataLen = 4
)

var (
	// ErrBadCommand is returned when a command request is invalid.
	ErrBadCommand = errors.New("bad command")

	// ErrBadLen is returned when a command doesn't have the expected length.
	ErrBadLen = errors.New("bad length")
)

// binWrite writes data to an io.Writer in big endian format.
func binWrite(w io.Writer, data ...interface{}) error {
	for _, v := range data {
		if err := binary.Write(w, binary.BigEndian, v); err != nil {
			return err
		}
	}
	return nil
}

// unmarshal32 unmarshals a message that's at most 32 bytes long,
// and has its length defined as the first 4 bytes.
func unmarshal32(b []byte) ([]byte, error) {
	if len(b) < DataLen {
		return nil, fmt.Errorf("%w: no data", ErrBadLen)
	}
	l := binary.BigEndian.Uint32(b[0:DataLen])
	data := b[DataLen:]
	if len(data) != int(l) {
		return nil, fmt.Errorf("%w: %v != %v", ErrBadLen, len(data), l)
	}
	return data, nil
}

// unmarshalAck reads an ack from the wire.
func unmarshalAck(b []byte) (uint64, error) {
	if len(b) != AckLen {
		return 0, fmt.Errorf("%w: %v != %v", ErrBadLen, len(b), AckLen)
	}
	return binary.BigEndian.Uint64(b[:AckLen]), nil
}

// Unmarshal creates a Command from a message in wire format.
func Unmarshal(b []byte) (Command, error) {
	if len(b) < TagLen {
		return nil, ErrBadCommand
	}
	tag := Tag(binary.BigEndian.Uint16(b[0:TagLen]))
	b = b[TagLen:]
	switch tag {
	case TagConnectSuccess:
		sid, err := unmarshal32(b)
		if err != nil {
			return nil, err
		}
		return NewConnectSuccess(sid)
	case TagReconnectSuccess:
		ack, err := unmarshalAck(b)
		if err != nil {
			return nil, err
		}
		return NewReconnectSuccess(ack), nil
	case TagData:
		data, err := unmarshal32(b)
		if err != nil {
			return nil, err
		}
		return NewData(data)
	case TagAck:
		ack, err := unmarshalAck(b)
		if err != nil {
			return nil, err
		}
		return NewAck(ack), nil
	default:
		return nil, ErrBadCommand
	}
}

// A Command is a corp-relay-v4@google.com command.
type Command interface {
	// Tag returns the command's tag.
	Tag() Tag

	// Write writes the command in wire format.
	Write(w io.Writer) error
}

// NewConnectSuccess creates a CONNECT_SUCCESS command with the given SID.
func NewConnectSuccess(sid []byte) (ConnectSuccess, error) {
	cs := ConnectSuccess(sid)
	return cs, cs.check()
}

// ConnectSuccess is the first command after a /v4/connect session is established.
// It is only sent from the server to the client.
type ConnectSuccess []byte

// Tag returns the command's tag.
func (cs ConnectSuccess) Tag() Tag {
	return TagConnectSuccess
}

// check verifies the command has a valid structure.
func (cs ConnectSuccess) check() error {
	if len(cs) < SIDLen {
		return fmt.Errorf("%w: SID length %v < %v", ErrBadLen, len(cs), SIDLen)
	}
	if len(cs) > MaxSIDLen {
		return fmt.Errorf("%w: SID length %v > %v", ErrBadLen, len(cs), MaxSIDLen)
	}
	return nil
}

// Write writes the command in wire format.
func (cs ConnectSuccess) Write(w io.Writer) error {
	if err := binWrite(w, cs.Tag(), uint32(len(cs))); err != nil {
		return err
	}
	_, err := w.Write(cs)
	return err
}

// SID returns the session ID.
// Even though the spec doesn't specify a format,
// https://chromium.googlesource.com/apps/libapps/+/HEAD/nassh/js/nassh_stream_relay_corpv4.js
// restricts us to ASCII SIDs.
func (cs ConnectSuccess) SID() string {
	return string(cs)
}

// NewReconnectSuccess creates a RECONNECT_SUCCESS command with the given ack.
func NewReconnectSuccess(ack uint64) ReconnectSuccess {
	return ReconnectSuccess(ack)
}

// ReconnectSuccess is the first command after a /v4/reconnect session is established.
// It is only sent from the server to the client.
type ReconnectSuccess uint64

// Tag returns the command's tag.
func (rs ReconnectSuccess) Tag() Tag {
	return TagReconnectSuccess
}

// Write writes the command in wire format.
func (rs ReconnectSuccess) Write(w io.Writer) error {
	return binWrite(w, rs.Tag(), rs)
}

// NewData creates a DATA command with the given payload.
func NewData(b []byte) (Data, error) {
	d := Data(b)
	return d, d.check()
}

// Data is arbitrary data sent on the wire.
type Data []byte

// Tag returns the command's tag.
func (d Data) Tag() Tag {
	return TagData
}

// check verifies the command has a valid structure.
func (d Data) check() error {
	if len(d) == 0 {
		return fmt.Errorf("%w: no data", ErrBadLen)
	}
	if len(d) > MaxArrayLen {
		return fmt.Errorf("%w: data too large %v > %v", ErrBadLen, len(d), MaxArrayLen)
	}
	return nil
}

// Write writes the command in wire format.
func (d Data) Write(w io.Writer) error {
	if err := binWrite(w, d.Tag(), uint32(len(d))); err != nil {
		return err
	}
	_, err := w.Write(d)
	return err
}

// Data returns the payload.
func (d Data) Data() []byte {
	return d
}

// NewAck creates an ACK command with the given ack.
func NewAck(ack uint64) Ack {
	return Ack(ack)
}

// Ack is an acknowledgement for received data.
// It indicates the receiving end should update its buffer to discard all acknowledged data.
type Ack uint64

// Tag returns the command's tag.
func (a Ack) Tag() Tag {
	return TagAck
}

// Write writes the command in wire format.
func (a Ack) Write(w io.Writer) error {
	return binWrite(w, a.Tag(), a)
}

// Ack returns the ack.
func (a Ack) Ack() uint64 {
	return uint64(a)
}
