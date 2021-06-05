package command

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/kylelemons/godebug/pretty"
)

var (
	csGood = []byte{
		0x00, 0x01, // tag
		0x00, 0x00, 0x00, 0x04, // length
		0xca, 0xfe, 0xbe, 0xef, // SID
	}
	csShort = []byte{
		0x00, 0x01, // tag
		0x00, 0x00, 0x00, 0xff, // length
		0xca, 0xfe, // SID
	}
	csLong = []byte{
		0x00, 0x01, // tag
		0x00, 0x00, 0x00, 0x04, // length
		0xca, 0xfe, 0xbe, 0xef, 0xf0, 0x0f, // SID
	}

	rsZeroAck = []byte{
		0x00, 0x02, // tag
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // ack
	}
	rsMaxAck = []byte{
		0x00, 0x02, // tag
		0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, // ack
	}
	rsShortAck = []byte{
		0x00, 0x02, // tag
		0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, // ack
	}
	rsLongAck = []byte{
		0x00, 0x02, // tag
		0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, // ack
	}

	dataGood = []byte{
		0x00, 0x04, // tag
		0x00, 0x00, 0x00, 0x10, // length
		0xca, 0xfe, 0xbe, 0xef, 0xca, 0xfe, 0xbe, 0xef, // data
		0xca, 0xfe, 0xbe, 0xef, 0xca, 0xfe, 0xbe, 0xef, // data
	}
	dataShort = []byte{
		0x00, 0x04, // tag
		0x00, 0x00, 0x00, 0x04, // length
		0xca, 0xfe, 0xbe, // data
	}
	dataLong = []byte{
		0x00, 0x04, // tag
		0x00, 0x00, 0x00, 0x04, // length
		0xca, 0xfe, 0xbe, 0xef, 0xca, 0xfe, 0xbe, 0xef, // data
	}
	dataZero = []byte{
		0x00, 0x04, // tag
	}

	ackZero = []byte{
		0x00, 0x07, // tag
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // ack
	}
	ackMax = []byte{
		0x00, 0x07, // tag
		0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, // ack
	}
	ackShort = []byte{
		0x00, 0x07, // tag
		0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, // ack
	}
	ackLong = []byte{
		0x00, 0x07, // tag
		0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, // ack
	}

	tagShort       = []byte{0x00}
	tagInvalid     = []byte{0x00, 0xff}
	unmarshalShort = []byte{0x00, 0x04}
)

type badWriter struct{}

func (badWriter) Write([]byte) (int, error) {
	return 0, errors.New("write error")
}

func TestTag(t *testing.T) {
	testdata := map[Tag]string{
		TagConnectSuccess:   "CONNECT_SUCCESS",
		TagReconnectSuccess: "RECONNECT_SUCCESS",
		TagData:             "DATA",
		TagAck:              "ACK",
		65535:               "unknown",
	}
	for tag, want := range testdata {
		if got := tag.String(); got != want {
			t.Errorf("String(%d) = %v, want %v", tag, got, want)
		}
	}
}

func TestUnmarshal(t *testing.T) {
	testdata := []struct {
		name string
		b    []byte
		tag  Tag
		ok   bool
	}{
		{
			name: "cs good",
			b:    csGood,
			tag:  TagConnectSuccess,
			ok:   true,
		},
		{
			name: "cs short",
			b:    csShort,
		},
		{
			name: "cs long",
			b:    csLong,
		},
		{
			name: "rs zero ack",
			b:    rsZeroAck,
			tag:  TagReconnectSuccess,
			ok:   true,
		},
		{
			name: "rs max ack",
			b:    rsMaxAck,
			tag:  TagReconnectSuccess,
			ok:   true,
		},
		{
			name: "rs short ack",
			b:    rsShortAck,
		},
		{
			name: "rs long ack",
			b:    rsLongAck,
		},
		{
			name: "data good",
			b:    dataGood,
			tag:  TagData,
			ok:   true,
		},
		{
			name: "data short",
			b:    dataShort,
		},
		{
			name: "data long",
			b:    dataLong,
		},
		{
			name: "data zero",
			b:    dataZero,
		},
		{
			name: "ack zero",
			b:    ackZero,
			tag:  TagAck,
			ok:   true,
		},
		{
			name: "max ack",
			b:    ackMax,
			tag:  TagAck,
			ok:   true,
		},
		{
			name: "ack short",
			b:    ackShort,
		},
		{
			name: "ack long",
			b:    ackLong,
		},
		{
			name: "tag short",
			b:    tagShort,
		},
		{
			name: "tag invalid",
			b:    tagInvalid,
		},
		{
			name: "unmarshal short",
			b:    unmarshalShort,
		},
	}
	for _, tt := range testdata {
		// Copy to ensure unintended mutations don't affect the golden data.
		want := make([]byte, len(tt.b))
		copy(want, tt.b)

		got, err := Unmarshal(tt.b)
		if err != nil {
			if tt.ok {
				t.Errorf("Unmarshal(%v) error = %v", tt.name, err)
			}
			continue
		}
		if !tt.ok {
			t.Errorf("Unmarshal(%v) error = nil", tt.name)
		}
		if got.Tag() != tt.tag {
			t.Errorf("Unmarshal(%v) tag = %v, want %v", tt.name, got.Tag(), tt.tag)
		}
		w := new(bytes.Buffer)
		if err := got.Write(w); err != nil {
			t.Errorf("Write(%v) error = %v", tt.name, err)
		}
		if diff := pretty.Compare(w.Bytes(), want); diff != "" {
			t.Errorf("Write(%v) diff (-got +want):\n%v", tt.name, diff)
		}
	}
}

func TestNewConnectSuccess(t *testing.T) {
	testdata := []struct {
		name string
		sid  []byte
		ok   bool
	}{
		{
			name: "good",
			sid:  make([]byte, 4),
			ok:   true,
		},
		{
			name: "short sid",
			sid:  make([]byte, 3),
		},
		{
			name: "long sid",
			sid:  make([]byte, 37),
		},
	}
	for _, tt := range testdata {
		if _, err := NewConnectSuccess(tt.sid); err != nil {
			if tt.ok {
				t.Errorf("NewConnectSuccess(%v) error = %v", tt.name, err)
			}
			continue
		}
		if !tt.ok {
			t.Errorf("NewConnectSuccess(%v) error = nil", tt.name)
		}
	}
}

func TestNewData(t *testing.T) {
	testdata := []struct {
		name string
		b    []byte
		ok   bool
	}{
		{
			name: "good",
			b:    make([]byte, 4),
			ok:   true,
		},
		{
			name: "no data",
		},
		{
			name: "long data",
			b:    make([]byte, 16*1024+1),
		},
	}
	for _, tt := range testdata {
		if _, err := NewData(tt.b); err != nil {
			if tt.ok {
				t.Errorf("NewData(%v) error = %v", tt.name, err)
			}
			continue
		}
		if !tt.ok {
			t.Errorf("NewData(%v) error = nil", tt.name)
		}
	}
}

func TestWrite(t *testing.T) {
	testdata := map[string]Command{
		"cs":   ConnectSuccess{0xaa, 0xbb},
		"rs":   ReconnectSuccess(123),
		"data": Data{0x00, 0xff},
		"ack":  Ack(456),
	}
	for name, tt := range testdata {
		var w io.Writer
		// Good write.
		w = new(strings.Builder)
		if err := tt.Write(w); err != nil {
			t.Errorf("Write(%v) error = %v", name, err)
		}

		// Bad writer.
		w = new(badWriter)
		if err := tt.Write(w); err == nil {
			t.Errorf("Write(%v) error = nil", name)
		}
	}
}
