package cookie

import (
	"net/http"
	"testing"

	"github.com/kylelemons/godebug/pretty"
)

func TestNew(t *testing.T) {
	testdata := []struct {
		uri  string
		want *Request
		ok   bool
	}{
		{
			uri: "/cookie?ext=foo&path=/&version=2&method=js-redirect",
			want: &Request{
				Ext:     "foo",
				Path:    "/",
				Version: 2,
				Method:  JSRedirect,
			},
			ok: true,
		},
		{
			uri: "/cookie?ext=foo&path=/&version=2&method=direct",
			want: &Request{
				Ext:     "foo",
				Path:    "/",
				Version: 2,
				Method:  Direct,
			},
			ok: true,
		},
		{
			uri: "/cookie?ext=foo&path=/",
			want: &Request{
				Ext:     "foo",
				Path:    "/",
				Version: 1,
				Method:  HTTPRedirect,
			},
			ok: true,
		},
		{
			uri: "/cookie?ext=foo&path=/&version=1",
			want: &Request{
				Ext:     "foo",
				Path:    "/",
				Version: 1,
				Method:  HTTPRedirect,
			},
			ok: true,
		},
		// Explicit zero version.
		{
			uri: "/cookie?ext=foo&path=/&version=0",
		},
		// Negative version.
		{
			uri: "/cookie?ext=foo&path=/&version=-1",
		},
		// Invalid version.
		{
			uri: "/cookie?ext=foo&path=/&version=foo",
		},
		// Invalid method.
		{
			uri: "/cookie?ext=foo&path=/&method=invalid",
		},
		// V2 must specify method.
		{
			uri: "/cookie?ext=foo&path=/&version=2",
		},
		// V1 must not specify method.
		{
			uri: "/cookie?ext=foo&path=/&method=direct",
		},
		// No ext.
		{
			uri: "/cookie?path=/",
		},
		// No path.
		{
			uri: "/cookie?ext=foo",
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
