package handler

import (
	"encoding/base64"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/hazaelsan/ssh-relay/cookie-server/request/cookie"
	"github.com/hazaelsan/ssh-relay/response"
	"github.com/kylelemons/godebug/pretty"

	configpb "github.com/hazaelsan/ssh-relay/cookie-server/proto/v1/config_go_proto"
	cookiepb "github.com/hazaelsan/ssh-relay/proto/v1/cookie_go_proto"
)

type responseWriter interface {
	http.ResponseWriter
	Result() *http.Response
}

// badWriter is a helper to simulate Write() failures.
type badWriter struct {
	*httptest.ResponseRecorder
}

func (b *badWriter) Write([]byte) (int, error) {
	return 0, errors.New("Write() failed")
}

func jsRedir(redir string) string {
	enc := base64.URLEncoding.EncodeToString([]byte(redir))
	return fmt.Sprintf(`<!DOCTYPE html>
<html>
	<head>
		<meta charset="UTF-8" />
		<script>window.location.href = "chrome-extension://foo/path#%v";</script>
	</head>
	<body></body>
</html>`, enc)
}

func TestWriteResponse(t *testing.T) {
	testdata := []struct {
		resp *response.Response
		req  *cookie.Request
		w    responseWriter
		want string
		ok   bool
	}{
		{
			resp: &response.Response{Endpoint: "foo"},
			req: &cookie.Request{
				Ext:  "foo",
				Path: "path",
			},
			w:    httptest.NewRecorder(),
			want: jsRedir(`{"endpoint":"foo"}`),
			ok:   true,
		},
		{
			resp: &response.Response{Error: "foo"},
			req: &cookie.Request{
				Ext:  "foo",
				Path: "path",
			},
			w:    httptest.NewRecorder(),
			want: jsRedir(`{"endpoint":"","error":"foo"}`),
			ok:   true,
		},
		// Write failure.
		{
			resp: &response.Response{Endpoint: "foo"},
			req: &cookie.Request{
				Ext:  "foo",
				Path: "path",
			},
			w: &badWriter{httptest.NewRecorder()},
		},
	}
	for i, tt := range testdata {
		h := &Handler{
			req: tt.req,
			r:   httptest.NewRequest("GET", "/foo", nil),
			w:   tt.w,
		}
		if err := h.writeResponse(tt.resp); err != nil {
			if tt.ok {
				t.Errorf("writeResponse(%v) error = %v", i, err)
			}
			continue
		}
		if !tt.ok {
			t.Errorf("writeResponse(%v) error = nil", i)
		}
		got, err := ioutil.ReadAll(tt.w.Result().Body)
		if err != nil {
			t.Errorf("ioutil.ReadAll(%v) error = %v", i, err)
			continue
		}
		if string(got) != tt.want {
			t.Errorf("writeResponse(%v) = %v, want %v", i, string(got), tt.want)
		}
	}
}

func TestErr(t *testing.T) {
	testdata := []struct {
		version  int
		msg      string
		code     int
		w        responseWriter
		wantMsg  string
		wantCode int
	}{
		{
			version:  2,
			msg:      "foo bar baz",
			code:     500,
			w:        httptest.NewRecorder(),
			wantMsg:  jsRedir(`{"endpoint":"","error":"foo bar baz"}`),
			wantCode: 200,
		},
		{
			version:  1,
			msg:      "foo bar baz",
			code:     500,
			w:        httptest.NewRecorder(),
			wantMsg:  "foo bar baz\n",
			wantCode: 500,
		},
		// Write failures.
		{
			version:  2,
			msg:      "foo bar baz",
			code:     401,
			w:        &badWriter{httptest.NewRecorder()},
			wantCode: 401,
		},
		{
			version:  1,
			msg:      "foo bar baz",
			code:     401,
			w:        &badWriter{httptest.NewRecorder()},
			wantCode: 401,
		},
	}
	for _, tt := range testdata {
		h := &Handler{
			req: &cookie.Request{
				Ext:     "foo",
				Path:    "path",
				Version: tt.version,
			},
			r: httptest.NewRequest("GET", "/foo", nil),
			w: tt.w,
		}
		h.err(tt.msg, tt.code)
		resp := tt.w.Result()
		if resp.StatusCode != tt.wantCode {
			t.Errorf("err(%v, %v) code = %v, want %v", tt.msg, tt.code, resp.StatusCode, tt.wantCode)
		}
		got, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			t.Errorf("ioutil.ReadAll(%v, %v) error = %v", tt.msg, tt.code, err)
			continue
		}
		if string(got) != tt.wantMsg {
			t.Errorf("err(%v, %v) body = %v, want %v", tt.msg, tt.code, string(got), tt.wantMsg)
		}
	}
}

func TestSetCookies(t *testing.T) {
	want := []*http.Cookie{
		&http.Cookie{
			Name:     "cookie",
			Value:    "chrome-extension://foo",
			Path:     "/",
			Domain:   "example.org",
			MaxAge:   3,
			Secure:   true,
			HttpOnly: true,
			Raw:      "cookie=chrome-extension://foo; Path=/; Domain=example.org; Max-Age=3; HttpOnly; Secure",
		},
	}
	w := httptest.NewRecorder()
	h := &Handler{
		cfg: &configpb.Config{
			OriginCookie: &cookiepb.Cookie{
				Name:   "cookie",
				Path:   "/",
				Domain: ".example.org",
			},
		},
		req:    &cookie.Request{Ext: "foo"},
		maxAge: 3 * time.Second,
		w:      w,
	}
	h.setCookies()
	if diff := pretty.Compare(w.Result().Cookies(), want); diff != "" {
		t.Errorf("setCookies() diff (-got +want):\n%v", diff)
	}
}

func TestRedirectHTTP(t *testing.T) {
	wantLocation := []string{"chrome-extension://foo/bar#anonymous@relay.example.org:8022"}
	wantCookies := []*http.Cookie{
		&http.Cookie{
			Name:     "cookie",
			Value:    "chrome-extension://foo",
			Path:     "/",
			Domain:   "example.org",
			MaxAge:   3,
			Secure:   true,
			HttpOnly: true,
			Raw:      "cookie=chrome-extension://foo; Path=/; Domain=example.org; Max-Age=3; HttpOnly; Secure",
		},
	}
	wantCode := http.StatusSeeOther
	w := httptest.NewRecorder()
	h := &Handler{
		cfg: &configpb.Config{
			OriginCookie: &cookiepb.Cookie{
				Name:   "cookie",
				Path:   "/",
				Domain: ".example.org",
			},
			FallbackRelayHost: "relay.example.org:8022",
		},
		req: &cookie.Request{
			Ext:  "foo",
			Path: "bar",
		},
		maxAge: 3 * time.Second,
		r:      httptest.NewRequest("GET", "/foo", nil),
		w:      w,
	}
	if err := h.redirectHTTP(); err != nil {
		t.Fatalf("redirectHTTP() error = %v", err)
	}
	if diff := pretty.Compare(w.Result().Cookies(), wantCookies); diff != "" {
		t.Errorf("redirectHTTP() cookies diff (-got +want):\n%v", diff)
	}
	if got := w.Result().StatusCode; got != wantCode {
		t.Errorf("redirectHTTP() StatusCode = %v, want %v", got, wantCode)
	}
	if diff := pretty.Compare(w.Result().Header["Location"], wantLocation); diff != "" {
		t.Errorf("redirectHTTP() Location diff (-got +want):\n%v, diff", diff)
	}
}

func TestRedirectJS(t *testing.T) {
	wantBody := jsRedir(`{"endpoint":"relay.example.org:8022"}`)
	wantCookies := []*http.Cookie{
		&http.Cookie{
			Name:     "cookie",
			Value:    "chrome-extension://foo",
			Path:     "/",
			Domain:   "example.org",
			MaxAge:   3,
			Secure:   true,
			HttpOnly: true,
			Raw:      "cookie=chrome-extension://foo; Path=/; Domain=example.org; Max-Age=3; HttpOnly; Secure",
		},
	}
	w := httptest.NewRecorder()
	h := &Handler{
		cfg: &configpb.Config{
			OriginCookie: &cookiepb.Cookie{
				Name:   "cookie",
				Path:   "/",
				Domain: ".example.org",
			},
			FallbackRelayHost: "relay.example.org:8022",
		},
		req: &cookie.Request{
			Ext:  "foo",
			Path: "path",
		},
		maxAge: 3 * time.Second,
		r:      httptest.NewRequest("GET", "/foo", nil),
		w:      w,
	}
	if err := h.redirectJS(); err != nil {
		t.Fatalf("redirectJS() error = %v", err)
	}
	if diff := pretty.Compare(w.Result().Cookies(), wantCookies); diff != "" {
		t.Errorf("redirectJS() cookies diff (-got +want):\n%v", diff)
	}
	got, err := ioutil.ReadAll(w.Result().Body)
	if err != nil {
		t.Fatalf("ioutil.ReadAll() error = %v", err)
	}
	if string(got) != wantBody {
		t.Errorf("redirectJS() body = %v, want %v", string(got), wantBody)
	}
}

func TestRedirectXSSI(t *testing.T) {
	wantBody := ")]}'\n" + `{"endpoint":"relay.example.org:8022"}`
	wantMIME := []string{"application/json"}
	wantCookies := []*http.Cookie{
		&http.Cookie{
			Name:     "cookie",
			Value:    "chrome-extension://foo",
			Path:     "/",
			Domain:   "example.org",
			MaxAge:   3,
			Secure:   true,
			HttpOnly: true,
			Raw:      "cookie=chrome-extension://foo; Path=/; Domain=example.org; Max-Age=3; HttpOnly; Secure",
		},
	}
	w := httptest.NewRecorder()
	h := &Handler{
		cfg: &configpb.Config{
			OriginCookie: &cookiepb.Cookie{
				Name:   "cookie",
				Path:   "/",
				Domain: ".example.org",
			},
			FallbackRelayHost: "relay.example.org:8022",
		},
		req: &cookie.Request{
			Ext:  "foo",
			Path: "path",
		},
		maxAge: 3 * time.Second,
		r:      httptest.NewRequest("GET", "/foo", nil),
		w:      w,
	}
	if err := h.redirectXSSI(); err != nil {
		t.Fatalf("redirectXSSI() error = %v", err)
	}
	if diff := pretty.Compare(w.Result().Header["Content-Type"], wantMIME); diff != "" {
		t.Errorf("redirectHTTP() Content-Type diff (-got +want):\n%v, diff", diff)
	}
	if diff := pretty.Compare(w.Result().Cookies(), wantCookies); diff != "" {
		t.Errorf("redirectXSSI() cookies diff (-got +want):\n%v", diff)
	}
	got, err := ioutil.ReadAll(w.Result().Body)
	if err != nil {
		t.Fatalf("ioutil.ReadAll() error = %v", err)
	}
	if string(got) != wantBody {
		t.Errorf("redirectXSSI() body = %v, want %v", string(got), wantBody)
	}

}
