package response

import (
	"strings"
	"testing"

	"github.com/kylelemons/godebug/pretty"
)

func TestFromReader(t *testing.T) {
	testdata := []struct {
		msg  string
		want *Response
		ok   bool
	}{
		{
			msg:  `)]}'{"Endpoint": "foo"}`,
			want: &Response{Endpoint: "foo"},
			ok:   true,
		},
		{
			msg:  ")]}'\n" + `{"Endpoint": "foo"}`,
			want: &Response{Endpoint: "foo"},
			ok:   true,
		},
		// Bad header.
		{
			msg: `)]}')]}'{"Endpoint": "foo"}`,
		},
		// No header.
		{
			msg: `{"Endpoint": "foo"}`,
		},
		// Malformed JSON.
		{
			msg: `)]}'"Endpoint": "foo"`,
		},
	}

	for _, tt := range testdata {
		r := strings.NewReader(tt.msg)
		got, err := FromReader(r)
		if err != nil {
			if tt.ok {
				t.Errorf("FromReader(%v) error = %v", tt.msg, err)
			}
			continue
		}
		if !tt.ok {
			t.Errorf("FromReader(%v) error = nil", tt.msg)
		}

		if diff := pretty.Compare(got, tt.want); diff != "" {
			t.Errorf("FromReader(%v) diff (-got +want):\n%v", tt.msg, diff)
		}
	}
}
