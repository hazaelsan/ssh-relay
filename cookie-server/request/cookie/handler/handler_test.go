package handler

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/hazaelsan/ssh-relay/response"
	"github.com/kylelemons/godebug/pretty"
	"google.golang.org/grpc"

	dpb "github.com/golang/protobuf/ptypes/duration"
	configpb "github.com/hazaelsan/ssh-relay/cookie-server/proto/v1/config_go_proto"
	requestpb "github.com/hazaelsan/ssh-relay/cookie-server/proto/v1/request_go_proto"
	servicepb "github.com/hazaelsan/ssh-relay/cookie-server/proto/v1/service_go_proto"
	cookiepb "github.com/hazaelsan/ssh-relay/proto/v1/cookie_go_proto"
	statuspb "google.golang.org/genproto/googleapis/rpc/status"
)

type authServer struct {
	endpoint string
	uri      string
	status   *statuspb.Status
	err      error
}

func (a *authServer) Authorize(_ context.Context, req *servicepb.AuthorizeRequest, _ ...grpc.CallOption) (*servicepb.AuthorizeResponse, error) {
	if a.err != nil {
		return nil, a.err
	}
	resp := &servicepb.AuthorizeResponse{
		Method: req.GetRequest().GetMethod(),
		Status: a.status,
	}
	if a.endpoint != "" {
		resp.Redirect = &servicepb.AuthorizeResponse_Endpoint{a.endpoint}
	}
	if a.uri != "" {
		resp.Redirect = &servicepb.AuthorizeResponse_NextUri{a.uri}
	}
	return resp, nil
}

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

func TestNew(t *testing.T) {
	testdata := []struct {
		name string
		cfg  *configpb.Config
		want time.Duration
		ok   bool
	}{
		{
			name: "good",
			cfg: &configpb.Config{
				OriginCookie: &cookiepb.Cookie{
					MaxAge: &dpb.Duration{Seconds: 3},
				},
			},
			want: 3 * time.Second,
			ok:   true,
		},
		{
			name: "bad MaxAge",
			cfg: &configpb.Config{
				OriginCookie: &cookiepb.Cookie{
					MaxAge: &dpb.Duration{Seconds: -3},
				},
			},
		},
	}
	for _, tt := range testdata {
		h, err := New(nil, tt.cfg, nil, nil, nil)
		if err != nil {
			if tt.ok {
				t.Errorf("New(%v) error = %v", tt.name, err)
			}
			continue
		}
		if !tt.ok {
			t.Errorf("New(%v) error = nil", tt.name)
		}
		if h.maxAge != tt.want {
			t.Errorf("New(%v) maxAge = %v, want %v", tt.name, h.maxAge, tt.want)
		}
	}
}

func TestWriteResponse(t *testing.T) {
	testdata := []struct {
		name string
		resp *response.Response
		req  *requestpb.Request
		w    responseWriter
		want string
		ok   bool
	}{
		{
			name: "good",
			resp: &response.Response{Endpoint: "foo"},
			req: &requestpb.Request{
				Ext:  "foo",
				Path: "path",
			},
			w:    httptest.NewRecorder(),
			want: jsRedir(`{"endpoint":"foo"}`),
			ok:   true,
		},
		{
			name: "good error",
			resp: &response.Response{Error: "foo"},
			req: &requestpb.Request{
				Ext:  "foo",
				Path: "path",
			},
			w:    httptest.NewRecorder(),
			want: jsRedir(`{"endpoint":"","error":"foo"}`),
			ok:   true,
		},
		{
			name: "write error",
			resp: &response.Response{Endpoint: "foo"},
			req: &requestpb.Request{
				Ext:  "foo",
				Path: "path",
			},
			w: &badWriter{httptest.NewRecorder()},
		},
		{
			name: "encode error",
			resp: &response.Response{},
			req: &requestpb.Request{
				Ext:  "foo",
				Path: "path",
			},
			w: &badWriter{httptest.NewRecorder()},
		},
	}
	for _, tt := range testdata {
		h := &Handler{
			req: tt.req,
			r:   httptest.NewRequest("GET", "/foo", nil),
			w:   tt.w,
		}
		if err := h.writeResponse(tt.resp); err != nil {
			if tt.ok {
				t.Errorf("writeResponse(%v) error = %v", tt.name, err)
			}
			continue
		}
		if !tt.ok {
			t.Errorf("writeResponse(%v) error = nil", tt.name)
		}
		got, err := ioutil.ReadAll(tt.w.Result().Body)
		if err != nil {
			t.Errorf("ioutil.ReadAll(%v) error = %v", tt.name, err)
			continue
		}
		if diff := pretty.Compare(string(got), tt.want); diff != "" {
			t.Errorf("writeResponse(%v) diff (-got +want):\n%v", tt.name, diff)
		}
	}
}

func TestErr(t *testing.T) {
	testdata := []struct {
		version  int32
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
			req: &requestpb.Request{
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
		if diff := pretty.Compare(string(got), tt.wantMsg); diff != "" {
			t.Errorf("err(%v, %v) body diff (-got +want):\n%v", tt.msg, tt.code, diff)
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
			SameSite: http.SameSiteNoneMode,
			Raw:      "cookie=chrome-extension://foo; Path=/; Domain=example.org; Max-Age=3; HttpOnly; Secure; SameSite=None",
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
		req:    &requestpb.Request{Ext: "foo"},
		maxAge: 3 * time.Second,
		w:      w,
	}
	h.setCookies()
	if diff := pretty.Compare(w.Result().Cookies(), want); diff != "" {
		t.Errorf("setCookies() diff (-got +want):\n%v", diff)
	}
}

func TestRedirectHTTP(t *testing.T) {
	uri := "chrome-extension://foo/bar#anonymous@relay.example.org:8022"
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
			SameSite: http.SameSiteNoneMode,
			Raw:      "cookie=chrome-extension://foo; Path=/; Domain=example.org; Max-Age=3; HttpOnly; Secure; SameSite=None",
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
		},
		req: &requestpb.Request{
			Ext:  "foo",
			Path: "bar",
		},
		maxAge: 3 * time.Second,
		r:      httptest.NewRequest("GET", "/foo", nil),
		w:      w,
	}
	if err := h.redirectHTTP(uri); err != nil {
		t.Fatalf("redirectHTTP(%v) error = %v", uri, err)
	}
	if diff := pretty.Compare(w.Result().Cookies(), wantCookies); diff != "" {
		t.Errorf("redirectHTTP(%v) cookies diff (-got +want):\n%v", uri, diff)
	}
	if got := w.Result().StatusCode; got != wantCode {
		t.Errorf("redirectHTTP(%v) StatusCode = %v, want %v", uri, got, wantCode)
	}
	if diff := pretty.Compare(w.Result().Header["Location"], wantLocation); diff != "" {
		t.Errorf("redirectHTTP(%v) Location diff (-got +want):\n%v, diff", uri, diff)
	}
}

func TestRedirectJS(t *testing.T) {
	resp := &response.Response{Endpoint: "relay.example.org:8022"}
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
			SameSite: http.SameSiteNoneMode,
			Raw:      "cookie=chrome-extension://foo; Path=/; Domain=example.org; Max-Age=3; HttpOnly; Secure; SameSite=None",
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
		req: &requestpb.Request{
			Ext:  "foo",
			Path: "path",
		},
		maxAge: 3 * time.Second,
		r:      httptest.NewRequest("GET", "/foo", nil),
		w:      w,
	}
	if err := h.redirectJS(resp); err != nil {
		t.Fatalf("redirectJS(%v) error = %v", resp, err)
	}
	if diff := pretty.Compare(w.Result().Cookies(), wantCookies); diff != "" {
		t.Errorf("redirectJS(%v) cookies diff (-got +want):\n%v", resp, diff)
	}
	got, err := ioutil.ReadAll(w.Result().Body)
	if err != nil {
		t.Fatalf("ioutil.ReadAll() error = %v", err)
	}
	if diff := pretty.Compare(string(got), wantBody); diff != "" {
		t.Errorf("redirectJS(%v) body diff (-got +want):\n%v", resp, diff)
	}
}

func TestRedirectXSSI(t *testing.T) {
	resp := &response.Response{Endpoint: "relay.example.org:8022"}
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
			SameSite: http.SameSiteNoneMode,
			Raw:      "cookie=chrome-extension://foo; Path=/; Domain=example.org; Max-Age=3; HttpOnly; Secure; SameSite=None",
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
		req: &requestpb.Request{
			Ext:  "foo",
			Path: "path",
		},
		maxAge: 3 * time.Second,
		r:      httptest.NewRequest("GET", "/foo", nil),
		w:      w,
	}
	if err := h.redirectXSSI(resp); err != nil {
		t.Fatalf("redirectXSSI(%v) error = %v", resp, err)
	}
	if diff := pretty.Compare(w.Result().Header["Content-Type"], wantMIME); diff != "" {
		t.Errorf("redirectXSSI(%v) Content-Type diff (-got +want):\n%v, diff", resp, diff)
	}
	if diff := pretty.Compare(w.Result().Cookies(), wantCookies); diff != "" {
		t.Errorf("redirectXSSI(%v) cookies diff (-got +want):\n%v", resp, diff)
	}
	got, err := ioutil.ReadAll(w.Result().Body)
	if err != nil {
		t.Fatalf("ioutil.ReadAll() error = %v", err)
	}
	if string(got) != wantBody {
		t.Errorf("redirectXSSI(%v) body = %v, want %v", resp, string(got), wantBody)
	}
}

func TestHandle(t *testing.T) {
	testdata := []struct {
		name   string
		s      servicepb.CookieServerClient
		method requestpb.RedirectionMethod
		ok     bool
	}{
		{
			name:   "good endpoint direct",
			s:      &authServer{endpoint: "/foo"},
			method: requestpb.RedirectionMethod_DIRECT,
			ok:     true,
		},
		{
			name:   "good endpoint http redirect",
			s:      &authServer{endpoint: "/foo"},
			method: requestpb.RedirectionMethod_HTTP_REDIRECT,
			ok:     true,
		},
		{
			name:   "good endpoint js redirect",
			s:      &authServer{endpoint: "/foo"},
			method: requestpb.RedirectionMethod_JS_REDIRECT,
			ok:     true,
		},
		{
			name:   "good uri direct",
			s:      &authServer{uri: "uri"},
			method: requestpb.RedirectionMethod_DIRECT,
			ok:     true,
		},
		{
			name:   "good uri http redirect",
			s:      &authServer{uri: "uri"},
			method: requestpb.RedirectionMethod_HTTP_REDIRECT,
			ok:     true,
		},
		{
			name:   "good uri js redirect",
			s:      &authServer{uri: "uri"},
			method: requestpb.RedirectionMethod_JS_REDIRECT,
			ok:     true,
		},
		{
			name:   "no redirect",
			s:      new(authServer),
			method: requestpb.RedirectionMethod_HTTP_REDIRECT,
		},
		{
			name: "authorize error",
			s:    &authServer{err: errors.New("auth error")},
		},
		{
			name: "status error",
			s:    &authServer{status: &statuspb.Status{Code: 1, Message: "status error"}},
		},
		{
			name: "endpoint error",
			s:    &authServer{endpoint: "/foo"},
		},
		{
			name: "next uri error",
			s:    &authServer{uri: "uri"},
		},
	}
	for _, tt := range testdata {
		w := httptest.NewRecorder()
		cfg := &configpb.Config{
			OriginCookie: &cookiepb.Cookie{
				Name:   "cookie",
				Path:   "/",
				Domain: ".example.org",
			},
		}
		req := &requestpb.Request{
			Ext:    "foo",
			Path:   "path",
			Method: tt.method,
		}
		h, err := New(tt.s, cfg, req, w, httptest.NewRequest("GET", "/foo", nil))
		if err != nil {
			t.Errorf("New(%v) error = %v", tt.name, err)
			continue
		}
		if err := h.Handle(context.Background()); err != nil {
			if tt.ok {
				t.Errorf("Handle(%v) error = %v", tt.name, err)
			}
			continue
		}
		if !tt.ok {
			t.Errorf("Handle(%v) error = nil", tt.name)
		}
	}
}
