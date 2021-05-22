package request

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestOrigin(t *testing.T) {
	testdata := []struct {
		name   string
		cookie *http.Cookie
		origin string
		want   string
		ok     bool
	}{
		{
			name: "good",
			cookie: &http.Cookie{
				Name:  "foo",
				Value: "chrome-extension://foo",
			},
			origin: "foo",
			want:   "chrome-extension://foo",
			ok:     true,
		},
		{
			name: "bad cookie",
			cookie: &http.Cookie{
				Name:  "foo",
				Value: "invalid",
			},
			origin: "foo",
		},
		{
			name: "bad origin",
			cookie: &http.Cookie{
				Name:  "foo",
				Value: "chrome-extension://foo",
			},
			origin: "bar",
		},
		{
			name: "empty origin",
			cookie: &http.Cookie{
				Name:  "foo",
				Value: "chrome-extension://foo",
			},
		},
	}
	for _, tt := range testdata {
		req := httptest.NewRequest("GET", "/foo", nil)
		req.AddCookie(tt.cookie)
		got, err := Origin(req, tt.origin)
		if err != nil {
			if tt.ok {
				t.Errorf("Origin(%v, %v) error = %v", tt.name, tt.origin, err)
			}
			continue
		}
		if !tt.ok {
			t.Errorf("Origin(%v, %v) error = nil", tt.name, tt.origin)
		}
		if got != tt.want {
			t.Errorf("Origin(%v, %v) = %v, want %v", tt.name, tt.origin, got, tt.want)
		}
	}
}
