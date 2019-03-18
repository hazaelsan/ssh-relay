package request

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestOrigin(t *testing.T) {
	testdata := []struct {
		cookie *http.Cookie
		name   string
		want   string
		ok     bool
	}{
		{
			cookie: &http.Cookie{
				Name:  "foo",
				Value: "http://example.org",
			},
			name: "foo",
			want: "http://example.org",
			ok:   true,
		},
		{
			cookie: &http.Cookie{
				Name:  "foo",
				Value: "chrome-extension://foo",
			},
			name: "foo",
			want: "chrome-extension://foo",
			ok:   true,
		},
		// Bad cookie value.
		{
			cookie: &http.Cookie{
				Name:  "foo",
				Value: "invalid",
			},
			name: "foo",
		},
		// Bad cookie name.
		{
			cookie: &http.Cookie{
				Name:  "foo",
				Value: "chrome-extension://foo",
			},
			name: "bar",
		},
	}
	for i, tt := range testdata {
		req := httptest.NewRequest("GET", "/foo", nil)
		req.AddCookie(tt.cookie)
		got, err := Origin(req, tt.name)
		if err != nil {
			if tt.ok {
				t.Errorf("Origin(%v, %v) error = %v", i, tt.name, err)
			}
			continue
		}
		if !tt.ok {
			t.Errorf("Origin(%v, %v) error = nil", i, tt.name)
		}
		if got != tt.want {
			t.Errorf("Origin(%v, %v) = %v, want %v", i, tt.name, got, tt.want)
		}
	}
}
