package corprelayv4

import (
	"bytes"
	"io"
	"io/ioutil"
	"net"
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
			0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, // data
			0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, // data
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

type rwc struct {
	*bytes.Buffer
}

func (r *rwc) Close() error {
	return nil
}

func TestParseBinary(t *testing.T) {
	testdata := []struct {
		b       []byte
		sshWant []byte
		wsWant  []byte
		rc      uint64
		ok      bool
	}{
		{
			b: goodData(),
			sshWant: []byte{
				0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07,
				0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f,
			},
			wsWant: []byte{
				0x00, 0x07, // tag
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x10, // ack
			},
			rc: 16,
			ok: true,
		},
		{
			b:  zeroAck(),
			ok: true,
		},
		{
			b: shortAck(),
		},
		{
			b: goodCS(),
		},
		{
			b: zeroAckRS(),
		},
	}
	for i, tt := range testdata {
		a, b := net.Pipe()
		type channel struct {
			b   []byte
			err error
		}
		c := make(chan channel)
		go func(c chan<- channel) {
			defer a.Close()
			buf, err := ioutil.ReadAll(a)
			c <- channel{buf, err}
		}(c)
		ws := &rwc{new(bytes.Buffer)}
		s := New(b)
		s.wFunc = func(int) (io.WriteCloser, error) { return ws, nil }
		if ok := func() bool {
			defer b.Close()
			if err := s.parseBinary(bytes.NewBuffer(tt.b)); err != nil {
				if tt.ok {
					t.Errorf("parseBinary(%v) error = %v", i, err)
				}
				return false
			}
			return true
		}(); !ok {
			continue
		}
		if !tt.ok {
			t.Errorf("parseBinary(%v) error = nil", i)
			continue
		}
		if s.rCount != tt.rc {
			t.Errorf("parseBinary(%v) rCount = %v, want %v", i, s.rCount, tt.rc)
		}
		if diff := pretty.Compare(ws.Bytes(), tt.wsWant); diff != "" {
			t.Errorf("parseBinary(%v) WebSocket diff (-got +want):\n%v", i, diff)
		}

		got := <-c
		if got.err != nil {
			t.Errorf("ReadAll(%v) error = %v", i, got.err)
		}
		if diff := pretty.Compare(got.b, tt.sshWant); diff != "" {
			t.Errorf("parseBinary(%v) diff (-got +want):\n%v", i, diff)
		}
	}
}
