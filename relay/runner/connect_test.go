package runner

import (
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/hazaelsan/ssh-relay/relay/session"
	"github.com/hazaelsan/ssh-relay/relay/session/manager"
	"github.com/kylelemons/godebug/pretty"

	pb "github.com/hazaelsan/ssh-relay/relay/proto/config_go_proto"
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

func newRunner() *Runner {
	return &Runner{
		cfg: &pb.Config{
			OriginCookieName: "origin",
		},
		mgr: manager.New(1, maxAge),
	}
}

func newSSH(r *Runner) (net.Conn, *session.Session, error) {
	a, b := net.Pipe()
	s, err := r.mgr.New(b)
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
		if b, respErr := ioutil.ReadAll(resp.Body); respErr == nil {
			return 0, nil, fmt.Errorf("%v: %v", err, string(b))
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
	sid := s.SID.String()

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
	url := fmt.Sprintf("ws%v/connect?sid=%v&ack=0&pos=0&try=1", strings.TrimPrefix(srv.URL, "http"), s.SID)
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
