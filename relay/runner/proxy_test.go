package runner

import (
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/hazaelsan/ssh-relay/relay/session/manager"

	pb "github.com/hazaelsan/ssh-relay/relay/proto/config_go_proto"
)

const maxAge = 100 * time.Millisecond

func listener(done <-chan struct{}) (string, error) {
	s, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		return "", err
	}
	go func() {
		<-done
		s.Close()
	}()
	go func() {
		for {
			if _, err := s.Accept(); err != nil {
				return
			}
		}
	}()
	return strconv.Itoa(s.Addr().(*net.TCPAddr).Port), nil
}

func newRunner() *Runner {
	return &Runner{
		cfg: &pb.Config{
			OriginCookieName: "origin",
		},
		mgr: manager.New(1, maxAge),
	}
}

func testProxyHandle(r *Runner, req *http.Request) *http.Response {
	w := httptest.NewRecorder()
	r.proxyHandle(w, req)
	return w.Result()
}

func TestProxyHandle(t *testing.T) {
	// Failure modes.
	testdata := []struct {
		url      string
		cookies  []*http.Cookie
		wantCode int
	}{
		// Missing host.
		{
			url:      "/proxy?port=22",
			wantCode: http.StatusBadRequest,
		},
		// Missing port.
		{
			url:      "/proxy?host=localhost",
			wantCode: http.StatusBadRequest,
		},
		// Bad origin cookie.
		{
			url:      "/proxy?host=localhost&port=22",
			wantCode: http.StatusBadRequest,
		},
		// Bad origin value.
		{
			url: "/proxy?host=localhost&port=22",
			cookies: []*http.Cookie{
				&http.Cookie{
					Name:  "origin",
					Value: "invalid",
				},
			},
			wantCode: http.StatusBadRequest,
		},
		// Dial error.
		{
			url: "/proxy?host=invalid%20host&port=22",
			cookies: []*http.Cookie{
				&http.Cookie{
					Name:  "origin",
					Value: "chrome-extension://foo",
				},
			},
			wantCode: http.StatusBadGateway,
		},
	}
	for _, tt := range testdata {
		r := newRunner()
		req := httptest.NewRequest("GET", tt.url, nil)
		for _, c := range tt.cookies {
			req.AddCookie(c)
		}
		if got := testProxyHandle(r, req); got.StatusCode != tt.wantCode {
			t.Errorf("proxyHandle(%v) status code = %v, want %v", tt.url, got.StatusCode, tt.wantCode)
		}
	}

	p, _ := net.Pipe()
	defer p.Close()
	done := make(chan struct{})
	port, err := listener(done)
	defer close(done)
	r := newRunner()
	if _, err := r.mgr.New(p); err != nil {
		t.Errorf("mgr.New() error = %v", err)
	}
	url := fmt.Sprintf("/proxy?host=localhost&port=%v", port)
	req := httptest.NewRequest("GET", url, nil)
	req.AddCookie(&http.Cookie{
		Name:  "origin",
		Value: "chrome-extension://foo",
	})

	// Test session limit.
	wantCode := http.StatusServiceUnavailable
	if got := testProxyHandle(r, req); got.StatusCode != wantCode {
		t.Errorf("proxyHandle(%v) status code = %v, want %v", url, got.StatusCode, wantCode)
	}

	// Test session expiry.
	time.Sleep(2 * maxAge)
	wantCode = http.StatusOK
	got := testProxyHandle(r, req)
	if got.StatusCode != wantCode {
		t.Errorf("proxyHandle(%v) status code = %v, want %v", url, got.StatusCode, wantCode)
	}
	b, err := ioutil.ReadAll(got.Body)
	if err != nil {
		t.Errorf("ReadAll(%v) error = %v", url, err)
	}
	if _, err := uuid.ParseBytes(b); err != nil {
		t.Errorf("uuid.ParseBytes(%v) error = %v", string(b), err)
	}
}
