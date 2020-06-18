package cookie

import (
	"net/http"
	"testing"

	"github.com/kylelemons/godebug/pretty"

	requestpb "github.com/hazaelsan/ssh-relay/cookie-server/proto/v1/request_go_proto"
)

func TestNew(t *testing.T) {
	testdata := []struct {
		uri  string
		want *requestpb.Request
		ok   bool
	}{
		{
			uri: "/cookie?ext=foo&path=/&version=2&method=js-redirect",
			want: &requestpb.Request{
				Ext:     "foo",
				Path:    "/",
				Version: 2,
				Method:  requestpb.RedirectionMethod_JS_REDIRECT,
			},
			ok: true,
		},
		{
			uri: "/cookie?ext=foo&path=/&version=2&method=direct",
			want: &requestpb.Request{
				Ext:     "foo",
				Path:    "/",
				Version: 2,
				Method:  requestpb.RedirectionMethod_DIRECT,
			},
			ok: true,
		},
		{
			uri: "/cookie?ext=foo&path=/",
			want: &requestpb.Request{
				Ext:     "foo",
				Path:    "/",
				Version: 1,
				Method:  requestpb.RedirectionMethod_HTTP_REDIRECT,
			},
			ok: true,
		},
		{
			uri: "/cookie?ext=foo&path=/&version=1",
			want: &requestpb.Request{
				Ext:     "foo",
				Path:    "/",
				Version: 1,
				Method:  requestpb.RedirectionMethod_HTTP_REDIRECT,
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
