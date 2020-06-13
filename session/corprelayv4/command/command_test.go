package command

import (
	"bytes"
	"testing"

	"github.com/kylelemons/godebug/pretty"
)

var (
	goodCS = func() []byte {
		return []byte{
			0x00, 0x01, // tag
			0x00, 0x00, 0x00, 0x04, // length
			0xca, 0xfe, 0xbe, 0xef, // SID
		}
	}
	shortCS = func() []byte {
		return []byte{
			0x00, 0x01, // tag
			0x00, 0x00, 0x00, 0xff, // length
			0xca, 0xfe, // SID
		}
	}
	longCS = func() []byte {
		return []byte{
			0x00, 0x01, // tag
			0x00, 0x00, 0x00, 0x04, // length
			0xca, 0xfe, 0xbe, 0xef, 0xf0, 0x0f, // SID
		}
	}

	zeroAckRS = func() []byte {
		return []byte{
			0x00, 0x02, // tag
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // ack
		}
	}
	maxAckRS = func() []byte {
		return []byte{
			0x00, 0x02, // tag
			0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, // ack
		}
	}
	shortAckRS = func() []byte {
		return []byte{
			0x00, 0x02, // tag
			0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, // ack
		}
	}
	longAckRS = func() []byte {
		return []byte{
			0x00, 0x02, // tag
			0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, // ack
		}
	}

	goodData = func() []byte {
		return []byte{
			0x00, 0x04, // tag
			0x00, 0x00, 0x00, 0x10, // length
			0xca, 0xfe, 0xbe, 0xef, 0xca, 0xfe, 0xbe, 0xef, // data
			0xca, 0xfe, 0xbe, 0xef, 0xca, 0xfe, 0xbe, 0xef, // data
		}
	}
	shortData = func() []byte {
		return []byte{
			0x00, 0x04, // tag
			0x00, 0x00, 0x00, 0x04, // length
			0xca, 0xfe, 0xbe, // data
		}
	}
	longData = func() []byte {
		return []byte{
			0x00, 0x04, // tag
			0x00, 0x00, 0x00, 0x04, // length
			0xca, 0xfe, 0xbe, 0xef, 0xca, 0xfe, 0xbe, 0xef, // data
		}
	}

	zeroAck = func() []byte {
		return []byte{
			0x00, 0x07, // tag
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // ack
		}
	}
	maxAck = func() []byte {
		return []byte{
			0x00, 0x07, // tag
			0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, // ack
		}
	}
	shortAck = func() []byte {
		return []byte{
			0x00, 0x07, // tag
			0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, // ack
		}
	}
	longAck = func() []byte {
		return []byte{
			0x00, 0x07, // tag
			0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, // ack
		}
	}
)

func TestUnmarshal(t *testing.T) {
	testdata := []struct {
		b   []byte
		tag Tag
		ok  bool
	}{
		{
			b:   goodCS(),
			tag: TagConnectSuccess,
			ok:  true,
		},
		{
			b:   shortCS(),
			tag: TagConnectSuccess,
			ok:  false,
		},
		{
			b:   longCS(),
			tag: TagConnectSuccess,
			ok:  false,
		},
		{
			b:   zeroAckRS(),
			tag: TagReconnectSuccess,
			ok:  true,
		},
		{
			b:   maxAckRS(),
			tag: TagReconnectSuccess,
			ok:  true,
		},
		{
			b:   shortAckRS(),
			tag: TagReconnectSuccess,
			ok:  false,
		},
		{
			b:   longAckRS(),
			tag: TagReconnectSuccess,
			ok:  false,
		},
		{
			b:   goodData(),
			tag: TagData,
			ok:  true,
		},
		{
			b:   shortData(),
			tag: TagData,
			ok:  false,
		},
		{
			b:   longData(),
			tag: TagData,
			ok:  false,
		},
		{
			b:   zeroAck(),
			tag: TagAck,
			ok:  true,
		},
		{
			b:   maxAck(),
			tag: TagAck,
			ok:  true,
		},
		{
			b:   shortAck(),
			tag: TagAck,
			ok:  false,
		},
		{
			b:   longAck(),
			tag: TagAck,
			ok:  false,
		},
	}
	for _, tt := range testdata {
		// Copy to ensure unintended mutations don't affect the golden data.
		want := make([]byte, len(tt.b))
		copy(want, tt.b)

		got, err := Unmarshal(tt.b)
		if err != nil {
			if tt.ok {
				t.Errorf("Unmarshal(%v) error = %v / %v", tt.b, err, got)
			}
			continue
		}
		if !tt.ok {
			t.Errorf("Unmarshal(%v) error = nil / %v", tt.b, got)
			continue
		}
		if got.Tag() != tt.tag {
			t.Errorf("Unmarshal(%v) tag = %v, want %v", tt.b, got.Tag(), tt.tag)
		}
		w := new(bytes.Buffer)
		if err := got.Write(w); err != nil {
			t.Errorf("Write(%v) error = %v", tt.b, err)
		}
		if diff := pretty.Compare(w.Bytes(), want); diff != "" {
			t.Errorf("Write(%v) diff (-got +want):\n%v", tt.b, diff)
		}
	}
}
