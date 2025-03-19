// Package cookie implements functionality for interacting with the Cookie Server, see
// https://chromium.googlesource.com/apps/libapps/+/HEAD/nassh/docs/relay-protocol.md#corp-relay-cookie.
//
// NOTE: Only version 2 of the cookie protocol is supported.
package cookie

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/golang/glog"
	"github.com/hazaelsan/ssh-relay/helper/session"
	"github.com/hazaelsan/ssh-relay/response"
)

const (
	clientVersion = 2
	redirMethod   = "direct"
)

// Authenticate authenticates against the given Cookie Server,
// returns the relay address and cookies to use for the WebSocket session.
// NOTE: Only version 2 of the cookie protocol is supported.
func Authenticate(addr string, client *http.Client) (string, []*http.Cookie, error) {
	u := authURL(addr)
	glog.V(2).Infof("Authenticating against %v", u)
	resp, err := client.Get(u)
	if err != nil {
		return "", nil, err
	}
	defer resp.Body.Close()
	r, err := response.FromReader(resp.Body)
	if err != nil {
		return "", nil, fmt.Errorf("response.FromReader() error: %w", err)
	}
	return session.AddDefaultPort(r.Endpoint, session.DefaultPort), resp.Cookies(), nil
}

// authURL builds the correct URL for authenticating against the Cookie Server.
func authURL(addr string) string {
	u := url.URL{
		Scheme: "https",
		Host:   addr,
		Path:   "/cookie",
	}
	q := u.Query()
	q.Set("ext", session.ExtID)
	q.Set("path", "/") // Dummy path
	q.Set("version", strconv.Itoa(clientVersion))
	q.Set("method", redirMethod)
	u.RawQuery = q.Encode()
	return u.String()
}
