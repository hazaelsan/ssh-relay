package connect

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/kylelemons/godebug/pretty"
)

var sid = uuid.MustParse("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee")

func TestNew(t *testing.T) {
	testdata := []struct {
		uri  string
		want *Request
		ok   bool
	}{
		{
			uri: "/connect?ack=0&pos=0&try=1&sid=" + sid.String(),
			want: &Request{
				SID: sid,
				try: 1,
			},
			ok: true,
		},
		// Non-zero ack.
		{
			uri: "/connect?ack=1&pos=0&try=1&sid=" + sid.String(),
		},
		// Non-zero pos.
		{
			uri: "/connect?ack=0&pos=1&try=1&sid=" + sid.String(),
		},
		// Bad ack.
		{
			uri: "/connect?ack=foo&pos=0&try=1&sid=" + sid.String(),
		},
		// Bad pos.
		{
			uri: "/connect?ack=0&pos=foo&try=1&sid=" + sid.String(),
		},
		// Bad try.
		{
			uri: "/connect?ack=0&pos=0&try=foo&sid=" + sid.String(),
		},
		// Bad sid.
		{
			uri: "/connect?ack=0&pos=0&try=1&sid=foobar",
		},
	}
	for _, tt := range testdata {
		req, err := http.NewRequest("GET", tt.uri, nil)
		if err != nil {
			t.Errorf("http.NewRequest(%v) error = %v", tt.uri, err)
			continue
		}
		got, err := New(req)
		if err != nil {
			if tt.ok {
				t.Errorf("New(%v) error = %v", tt.uri, err)
			}
			continue
		}
		if !tt.ok {
			t.Errorf("New(%v) error = nil", tt.uri)
		}

		if diff := pretty.Compare(got, tt.want); diff != "" {
			t.Errorf("New(%v) diff (-got +want):\n%v", tt.uri, diff)
		}
	}
}
