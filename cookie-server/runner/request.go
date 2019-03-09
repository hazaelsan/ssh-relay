package runner

import (
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"time"

	"github.com/golang/glog"
	"github.com/hazaelsan/ssh-relay/duration"
	"github.com/hazaelsan/ssh-relay/request"
	"github.com/hazaelsan/ssh-relay/response"

	configpb "github.com/hazaelsan/ssh-relay/cookie-server/proto/config_go_proto"
	cookiepb "github.com/hazaelsan/ssh-relay/proto/cookie_go_proto"
)

const (
	extPrefix    = "chrome-extension://"
	extParam     = "ext"
	pathParam    = "path"
	versionParam = "version"
	methodParam  = "method"

	redirectTmpl = `
{{define "redir"}}
<!DOCTYPE html>
<html>
	<head>
		<meta charset="UTF-8" />
	<body>
		<script>window.location.href = "{{.}}"</script>
	</body>
</html>
{{end}}
`
)

// NewRequest creates a *Request from its HTTP counterparts.
func NewRequest(w http.ResponseWriter, req *http.Request, cfg *configpb.Config) (*Request, error) {
	cr := &CookieRequest{
		ext:     req.URL.Query().Get(extParam),
		path:    req.URL.Query().Get(pathParam),
		version: 1,
	}
	if cr.ext == "" || cr.path == "" {
		return nil, request.ErrBadRequest
	}
	v, err := strconv.Atoi(req.URL.Query().Get(versionParam))
	if err != nil || v == 0 {
		return nil, request.ErrBadRequest
	}
	cr.version = v
	m, ok := redirectionMethodMap[req.URL.Query().Get(methodParam)]
	if !ok {
		return nil, request.ErrBadRequest
	}
	cr.method = m
	r := &Request{
		w:   w,
		r:   req,
		cr:  cr,
		cfg: cfg,
	}
	if err := duration.FromProto(&r.maxAge, r.cfg.OriginCookie.MaxAge); err != nil {
		return nil, err
	}
	return r, nil
}

// A Request is a container around an HTTP request.
type Request struct {
	w      http.ResponseWriter
	r      *http.Request
	cr     *CookieRequest
	cfg    *configpb.Config
	maxAge time.Duration
}

// SetCookies sets all requisite cookies for redirection to work.
func (r *Request) SetCookies() {
	for c, val := range map[*cookiepb.Cookie]string{
		r.cfg.OriginCookie: extPrefix + r.cr.ext,
	} {
		http.SetCookie(r.w, r.cookie(c, val))
	}
}

// Error writes an error to the client, note that code is not used in v2 client responses.
func (r *Request) Error(msg string, code int) {
	if r.cr.version == 2 {
		if err := r.redirectJSON(response.FromError(msg)); err == nil {
			return
		}
	}
	// Fallback to plain HTTP status codes.
	http.Error(r.w, msg, code)
}

// cookie creates a cookie to send to a client.
func (r *Request) cookie(c *cookiepb.Cookie, val string) *http.Cookie {
	return &http.Cookie{
		Name:     c.Name,
		Value:    val,
		Path:     c.Path,
		Domain:   c.Domain,
		MaxAge:   int(r.maxAge.Seconds()),
		Secure:   true,
		HttpOnly: true,
	}
}

// redirectJSON returns a JavaScript redirect with a base64-encoded JSON response embedded in the URI fragment.
func (r *Request) redirectJSON(resp *response.Response) error {
	enc, err := resp.Encode()
	if err != nil {
		return err
	}
	redir := fmt.Sprintf("chrome-extension://%v/%v#%v", r.cr.ext, r.cr.path, enc)
	glog.V(2).Infof("Redirecting %v to %v", r.r.RemoteAddr, redir)
	t, err := template.New("html").Parse(redirectTmpl)
	if err != nil {
		return err
	}
	return t.ExecuteTemplate(r.w, "redir", redir)
}

// relay returns the address of the SSH-over-WebSocket Relay to use for a given client.
// TODO: Improve relay selection instead of a hardcoded address.
func (r *Request) relay() string {
	return r.cfg.FallbackRelayHost
}

// RedirectURI redirects v1 clients by sending a 303 redirect to user@relay_host:relay_port.
func (r *Request) RedirectURI() error {
	redir := fmt.Sprintf("%v%v/%v#%v@%v", extPrefix, r.cr.ext, r.cr.path, "anonymous", r.relay())
	glog.V(2).Infof("Redirecting %v to %v", r.r.RemoteAddr, redir)
	r.SetCookies()
	http.Redirect(r.w, r.r, redir, http.StatusSeeOther)
	return nil
}

// RedirectJS redirects v2 clients via a JavaScript redirect with a base64-encoded JSON response embedded in the URI fragment.
func (r *Request) RedirectJS() error {
	r.SetCookies()
	return r.redirectJSON(response.FromEndpoint(r.relay()))
}

// RedirectDirect redirects v2 clients by sending a JSON response with an XSSI header.
func (r *Request) RedirectDirect() error {
	b, err := response.FromEndpoint(r.relay()).MarshalXSSI()
	if err != nil {
		r.Error("internal server error", http.StatusInternalServerError)
		return err
	}
	r.SetCookies()
	r.w.Header().Set("Content-Type", "application/json")
	r.w.Write(b)
	return nil
}
