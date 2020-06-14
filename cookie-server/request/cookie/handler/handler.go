// Package handler implements an HTTP handler for /cookie requests.
package handler

import (
	"fmt"
	"html/template"
	"net/http"
	"time"

	"github.com/golang/glog"
	"github.com/hazaelsan/ssh-relay/cookie-server/request/cookie"
	"github.com/hazaelsan/ssh-relay/duration"
	"github.com/hazaelsan/ssh-relay/response"

	configpb "github.com/hazaelsan/ssh-relay/cookie-server/proto/v1/config_go_proto"
	cookiepb "github.com/hazaelsan/ssh-relay/proto/v1/cookie_go_proto"
)

const (
	extPrefix = "chrome-extension://"

	redirectTmpl = `<!DOCTYPE html>
<html>
	<head>
		<meta charset="UTF-8" />
		<script>window.location.href = {{.}};</script>
	</head>
	<body></body>
</html>`
)

// New creates a *Handler for an HTTP request.
func New(cfg *configpb.Config, req *cookie.Request, w http.ResponseWriter, r *http.Request) (*Handler, error) {
	h := &Handler{
		cfg: cfg,
		req: req,
		w:   w,
		r:   r,
	}
	if err := duration.FromProto(&h.maxAge, h.cfg.OriginCookie.MaxAge); err != nil {
		return nil, err
	}
	return h, nil
}

// A Handler is an HTTP handler for /cookie requests.
type Handler struct {
	cfg    *configpb.Config
	req    *cookie.Request
	maxAge time.Duration
	w      http.ResponseWriter
	r      *http.Request
}

// Handle processes the /cookie HTTP request, redirecting clients according to the configured method.
func (h *Handler) Handle() error {
	switch h.req.Method {
	case cookie.HTTPRedirect:
		return h.redirectHTTP()
	case cookie.Direct:
		return h.redirectXSSI()
	case cookie.JSRedirect:
		return h.redirectJS()
	default:
		return fmt.Errorf("bad redirection method: %v", h.req.Method)
	}
}

// writeResponse sends a JSON response as a base64-encoded URI fragment as a JavaScript redirect.
func (h *Handler) writeResponse(resp *response.Response) error {
	enc, err := resp.Encode()
	if err != nil {
		return err
	}
	t, err := template.New("html").Parse(redirectTmpl)
	if err != nil {
		return err
	}
	redir := fmt.Sprintf("%v%v/%v#%v", extPrefix, h.req.Ext, h.req.Path, enc)
	glog.V(4).Infof("Redirecting %v to %v %+v", h.r.RemoteAddr, redir, *resp)
	return t.Execute(h.w, redir)
}

// err writes an error to the client, note that code is not used in JSON responses.
func (h *Handler) err(msg string, code int) {
	if h.req.Version == 2 {
		if err := h.writeResponse(response.FromError(msg)); err == nil {
			return
		}
	}
	// Fallback to plain HTTP status codes.
	http.Error(h.w, msg, code)
}

// cookie creates a cookie to send to a client.
func (h *Handler) cookie(c *cookiepb.Cookie, val string) *http.Cookie {
	return &http.Cookie{
		Name:     c.Name,
		Value:    val,
		Path:     c.Path,
		Domain:   c.Domain,
		MaxAge:   int(h.maxAge.Seconds()),
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteNoneMode,
	}
}

// setCookies sets all requisite cookies for redirection to work.
func (h *Handler) setCookies() {
	for c, val := range map[*cookiepb.Cookie]string{
		h.cfg.OriginCookie: extPrefix + h.req.Ext,
	} {
		http.SetCookie(h.w, h.cookie(c, val))
	}
}

// relay returns the address of the SSH Relay to use for a given client.
// TODO: Improve relay selection instead of a hardcoded address.
func (h *Handler) relay() string {
	return h.cfg.FallbackRelayHost
}

// redirectHTTP redirects clients by sending an HTTP redirect.
func (h *Handler) redirectHTTP() error {
	redir := fmt.Sprintf("%v%v/%v#%v@%v", extPrefix, h.req.Ext, h.req.Path, "anonymous", h.relay())
	glog.V(4).Infof("Redirecting %v to %v", h.r.RemoteAddr, redir)
	h.setCookies()
	http.Redirect(h.w, h.r, redir, http.StatusSeeOther)
	return nil
}

// redirectJS redirects clients via a JavaScript redirect with a base64-encoded JSON response embedded in the URI fragment.
func (h *Handler) redirectJS() error {
	h.setCookies()
	return h.writeResponse(response.FromEndpoint(h.relay()))
}

// redirectXSSI redirects clients by sending a JSON response with an XSSI header.
func (h *Handler) redirectXSSI() error {
	b, err := response.FromEndpoint(h.relay()).MarshalXSSI()
	if err != nil {
		h.err(http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return err
	}
	h.setCookies()
	h.w.Header().Set("Content-Type", "application/json")
	_, err = h.w.Write(b)
	return err
}
