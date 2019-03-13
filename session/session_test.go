package session

import (
	"bytes"
	"io/ioutil"
	"testing"

	"github.com/kylelemons/godebug/pretty"
)

type rwc struct {
	*bytes.Buffer
}

func (r *rwc) Close() error {
	return nil
}

func TestIncCounter(t *testing.T) {
	testdata := []struct {
		c    uint32
		n    int
		want uint32
	}{
		{
			c:    1,
			n:    10,
			want: 11,
		},
		{
			c:    0xffffff,
			n:    10,
			want: 9,
		},
		{
			c:    0xffffff,
			n:    0x1000000,
			want: 0xffffff,
		},
	}
	for _, tt := range testdata {
		s := &Session{c: tt.c}
		s.incCounter(tt.n)
		if got := s.c; got != tt.want {
			t.Errorf("incCounter(%v, %v) = %v, want %v", tt.c, tt.n, got, tt.want)
		}
	}
}

func TestParseBinary(t *testing.T) {
	testdata := []struct {
		data []byte
		want []byte
		c    uint32
		ok   bool
	}{
		{
			data: []byte{0x00, 0x00, 0x00, 0x00, 0xaa, 0xbb},
			want: []byte{0xaa, 0xbb},
			c:    2,
			ok:   true,
		},
		{
			data: []byte{0x00, 0xff, 0xff, 0xff, 0xaa, 0xbb, 0xcc, 0xdd},
			want: []byte{0xaa, 0xbb, 0xcc, 0xdd},
			c:    4,
			ok:   true,
		},
		// Valid ack, no data.
		{
			data: []byte{0x00, 0x10, 0x20, 0x30},
			ok:   true,
		},
		// Error ack.
		{
			data: []byte{0x01, 0x00, 0x00, 0x00, 0xaa, 0xbb},
		},
		// Short read.
		{
			data: []byte{0x00, 0x01, 0x02},
		},
	}
	for i, tt := range testdata {
		r := bytes.NewReader(tt.data)
		s := &Session{
			ssh: &rwc{new(bytes.Buffer)},
		}
		if err := s.parseBinary(r); err != nil {
			if tt.ok {
				t.Errorf("parseBinary(%v) error = %v", i, err)
			}
			continue
		}
		if !tt.ok {
			t.Errorf("parseBinary(%v) error = nil", i)
		}
		got, err := ioutil.ReadAll(s.ssh)
		if err != nil {
			t.Errorf("ReadAll(%v) error = %v", i, err)
		}
		if diff := pretty.Compare(got, tt.want); diff != "" {
			t.Errorf("parseBinary(%v) diff (-got +want):\n%v", i, diff)
		}
		if s.c != tt.c {
			t.Errorf("parseBinary(%v) counter = %v, want %v", i, s.c, tt.c)
		}
	}
}

func TestReadAck(t *testing.T) {
	testdata := []struct {
		data []byte
		want uint32
		ok   bool
	}{
		{
			data: []byte{0x00, 0x00, 0x00, 0x00, 0xaa, 0xbb},
			want: 0,
			ok:   true,
		},
		{
			data: []byte{0x00, 0x10, 0x20, 0x30},
			want: 0x102030,
			ok:   true,
		},
		// High byte non-zero.
		{
			data: []byte{0x01, 0x02, 0x03, 0x04, 0xaa},
		},
		// Short read.
		{
			data: []byte{0x01, 0x02, 0x03},
		},
	}
	for i, tt := range testdata {
		r := bytes.NewReader(tt.data)
		got, err := readAck(r)
		if err != nil {
			if tt.ok {
				t.Errorf("readAck(%v) error = %v", i, err)
			}
			continue
		}
		if !tt.ok {
			t.Errorf("readAck(%v) error = nil", i)
		}
		if got != tt.want {
			t.Errorf("readAck(%v) = %v, want %v", i, got, tt.want)
		}
	}
}

func TestWriteAck(t *testing.T) {
	testdata := map[uint32][]byte{
		0:          []byte{0x00, 0x00, 0x00, 0x00},
		0xffffff:   []byte{0x00, 0xff, 0xff, 0xff},
		0x12345678: []byte{0x12, 0x34, 0x56, 0x78},
	}
	for ack, want := range testdata {
		w := new(bytes.Buffer)
		if err := writeAck(w, ack); err != nil {
			t.Errorf("writeAck(%v) error = %v", ack, err)
			continue
		}
		if diff := pretty.Compare(w.Bytes(), want); diff != "" {
			t.Errorf("writeAck(%v) diff (-got +want):\n%v", ack, diff)
		}
	}
}
