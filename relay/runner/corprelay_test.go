package runner

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/hazaelsan/ssh-relay/relay/session/manager"
	"github.com/hazaelsan/ssh-relay/session"
	"github.com/kylelemons/godebug/pretty"

	"github.com/hazaelsan/ssh-relay/relay/proto/v1/configpb"
)

const (
	maxAge   = time.Second
	dummySID = "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
)

var (
	originCookie = &http.Cookie{
		Name:  "origin",
		Value: "chrome-extension://foo",
	}
	sshMsg = []byte{0xab, 0xcd, 0xef}
)

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
		cfg: &configpb.Config{
			OriginCookieName: "origin",
		},
		mgr: manager.New(1, maxAge),
	}
}

func newSSH(r *Runner) (net.Conn, session.Session, error) {
	a, b := net.Pipe()
	s, err := r.mgr.New(b, session.CorpRelay)
	if err != nil {
		return nil, nil, err
	}
	return a, s, nil
}

func testConnectHandle(r *Runner, req *http.Request) *http.Response {
	w := httptest.NewRecorder()
	r.connectHandle(w, req)
	return w.Result()
}

func wsReq(url string) (int, []byte, error) {
	hdr := http.Header{}
	hdr.Add("Origin", "chrome-extension://foo")
	hdr.Add("Cookie", originCookie.String())
	ws, resp, err := websocket.DefaultDialer.Dial(url, hdr)
	if err != nil {
		if b, respErr := io.ReadAll(resp.Body); respErr == nil {
			return 0, nil, fmt.Errorf("%w: %v", err, string(b))
		}
		return 0, nil, err
	}
	defer ws.Close()
	return ws.ReadMessage()
}

func TestConnectHandle(t *testing.T) {
	r := newRunner()
	conn, s, err := newSSH(r)
	if err != nil {
		t.Fatalf("newSSH() error = %v", err)
	}
	sid := s.SID().String()

	// Failure modes.
	testdata := []struct {
		url      string
		wantCode int
	}{
		// Bad SID.
		{
			url:      "/connect?&ack=0&pos=0&try=1",
			wantCode: http.StatusBadRequest,
		},
		{
			url:      "/connect?sid=invalid&ack=0&pos=0&try=1",
			wantCode: http.StatusBadRequest,
		},
		{
			url:      fmt.Sprintf("/connect?sid=%v&ack=0&pos=0&try=1", dummySID),
			wantCode: http.StatusBadRequest,
		},
		// Bad ack.
		{
			url:      fmt.Sprintf("/connect?sid=%v&pos=0&try=1", sid),
			wantCode: http.StatusBadRequest,
		},
		{
			url:      fmt.Sprintf("/connect?sid=%v&ack=1&pos=0&try=1", sid),
			wantCode: http.StatusBadRequest,
		},
		// Bad pos.
		{
			url:      fmt.Sprintf("/connect?sid=%v&ack=0&try=1", sid),
			wantCode: http.StatusBadRequest,
		},
		{
			url:      fmt.Sprintf("/connect?sid=%v&ack=0&pos=1&try=1", sid),
			wantCode: http.StatusBadRequest,
		},
		// Bad try.
		{
			url:      fmt.Sprintf("/connect?sid=%v&ack=0&pos=0", sid),
			wantCode: http.StatusBadRequest,
		},
		// Non-WebSocket session.
		{
			url:      fmt.Sprintf("/connect?sid=%v&ack=0&pos=0&try=1", sid),
			wantCode: http.StatusBadRequest,
		},
	}
	for _, tt := range testdata {
		r := newRunner()
		req := httptest.NewRequest("GET", tt.url, nil)
		req.AddCookie(originCookie)
		if got := testConnectHandle(r, req); got.StatusCode != tt.wantCode {
			t.Errorf("connectHandle(%v) status code = %v, want %v", tt.url, got.StatusCode, tt.wantCode)
		}
	}

	wantMsg := []byte{0, 0, 0, 0, 0xab, 0xcd, 0xef}
	go conn.Write(sshMsg)
	srv := httptest.NewServer(http.HandlerFunc(r.connectHandle))
	url := fmt.Sprintf("ws%v/connect?sid=%v&ack=0&pos=0&try=1", strings.TrimPrefix(srv.URL, "http"), s.SID())
	mt, got, err := wsReq(url)
	if err != nil {
		t.Fatalf("wsReq(%v) error = %v", url, err)
	}
	if mt != websocket.BinaryMessage {
		t.Errorf("websocket message type = %v, want %v", mt, websocket.BinaryMessage)
	}
	if diff := pretty.Compare(got, wantMsg); diff != "" {
		t.Errorf("connectHandle() diff (-got +want):\n%v", diff)
	}

	// Ensure session has timed out.
	time.Sleep(maxAge)
	if _, _, err := wsReq(url); err == nil {
		t.Errorf("wsReq(%v) error = nil", url)
	}
}

func testProxyHandle(r *Runner, req *http.Request) *http.Response {
	w := httptest.NewRecorder()
	r.proxyHandle(w, req)
	return w.Result()
}

func TestProxyHandle(t *testing.T) {
	p, _ := net.Pipe()
	defer p.Close()
	done := make(chan struct{})
	port, err := listener(done)
	defer close(done)
	r := newRunner()
	if _, err := r.mgr.New(p, session.CorpRelay); err != nil {
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
	b, err := io.ReadAll(got.Body)
	if err != nil {
		t.Errorf("io.ReadAll(%v) error = %v", url, err)
	}
	if _, err := uuid.ParseBytes(b); err != nil {
		t.Errorf("uuid.ParseBytes(%v) error = %v", string(b), err)
	}
}

func TestProxyHandle_Failures(t *testing.T) {
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
}
