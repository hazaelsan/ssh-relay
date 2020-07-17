// Package handler implements an HTTP handler for /cookie requests.
package handler

import (
	"context"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"time"

	"github.com/golang/glog"
	"github.com/hazaelsan/ssh-relay/duration"
	"github.com/hazaelsan/ssh-relay/response"
	"google.golang.org/grpc/status"

	configpb "github.com/hazaelsan/ssh-relay/cookie-server/proto/v1/config_go_proto"
	requestpb "github.com/hazaelsan/ssh-relay/cookie-server/proto/v1/request_go_proto"
	servicepb "github.com/hazaelsan/ssh-relay/cookie-server/proto/v1/service_go_proto"
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

var (
	errBadMethod  = errors.New("bad redirection method")
	errNoRedirect = errors.New("no redirect in response")
)

// New creates a *Handler for an HTTP request.
func New(c servicepb.CookieServerClient, cfg *configpb.Config, req *requestpb.Request, w http.ResponseWriter, r *http.Request) (*Handler, error) {
	h := &Handler{
		c:   c,
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
	c      servicepb.CookieServerClient
	cfg    *configpb.Config
	req    *requestpb.Request
	maxAge time.Duration
	w      http.ResponseWriter
	r      *http.Request
}

// Handle processes the /cookie HTTP request, redirecting clients according to the configured method.
func (h *Handler) Handle(ctx context.Context) error {
	req := &servicepb.AuthorizeRequest{Request: h.req}
	resp, err := h.c.Authorize(ctx, req)
	if err != nil {
		return fmt.Errorf("Authorize(%v) error: %w", req, err)
	}
	if err := status.ErrorProto(resp.GetStatus()); err != nil {
		return fmt.Errorf("Authorize(%v) error: %w", req, err)
	}
	switch resp.GetRedirect().(type) {
	case *servicepb.AuthorizeResponse_NextUri:
		return h.redirectURI(resp.GetNextUri(), resp.GetMethod())
	case *servicepb.AuthorizeResponse_Endpoint:
		return h.redirectEndpoint(resp.GetEndpoint(), resp.GetMethod())
	default:
		return errNoRedirect
	}
}

// writeResponse sends a JSON response as a base64-encoded URI fragment as a JavaScript redirect.
func (h *Handler) writeResponse(r *response.Response) error {
	enc, err := r.Encode()
	if err != nil {
		return err
	}
	t, err := template.New("html").Parse(redirectTmpl)
	if err != nil {
		return err
	}
	uri := fmt.Sprintf("%v%v/%v#%v", extPrefix, h.req.GetExt(), h.req.GetPath(), enc)
	glog.V(4).Infof("Redirecting %v to %v %+v", h.r.RemoteAddr, uri, *r)
	return t.Execute(h.w, uri)
}

// err writes an error to the client, note that code is not used in JSON responses.
func (h *Handler) err(msg string, code int) {
	if h.req.GetVersion() == 2 {
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
		h.cfg.OriginCookie: extPrefix + h.req.GetExt(),
	} {
		http.SetCookie(h.w, h.cookie(c, val))
	}
}

// redirectURI redirects clients to a URI.
func (h *Handler) redirectURI(uri string, method requestpb.RedirectionMethod) error {
	switch method {
	case requestpb.RedirectionMethod_HTTP_REDIRECT:
		return h.redirectHTTP(uri)
	case requestpb.RedirectionMethod_DIRECT:
		return h.redirectXSSI(response.FromEndpoint(uri))
	case requestpb.RedirectionMethod_JS_REDIRECT:
		return h.redirectJS(response.FromEndpoint(uri))
	}
	return errBadMethod
}

// redirectHTTP redirects clients by sending an HTTP redirect.
func (h *Handler) redirectHTTP(uri string) error {
	glog.V(4).Infof("Redirecting %v to %v", h.r.RemoteAddr, uri)
	h.setCookies()
	http.Redirect(h.w, h.r, uri, http.StatusSeeOther)
	return nil
}

// redirectEndpoint redirects clients to an SSH relay endpoint.
func (h *Handler) redirectEndpoint(endpoint string, method requestpb.RedirectionMethod) error {
	switch method {
	case requestpb.RedirectionMethod_HTTP_REDIRECT:
		uri := fmt.Sprintf("%v%v/%v#%v@%v", extPrefix, h.req.GetExt(), h.req.GetPath(), "anonymous", endpoint)
		return h.redirectHTTP(uri)
	case requestpb.RedirectionMethod_DIRECT:
		return h.redirectXSSI(response.FromEndpoint(endpoint))
	case requestpb.RedirectionMethod_JS_REDIRECT:
		return h.redirectJS(response.FromEndpoint(endpoint))
	}
	return errBadMethod
}

// redirectJS redirects clients via a JavaScript redirect with a base64-encoded JSON response embedded in the URI fragment.
func (h *Handler) redirectJS(r *response.Response) error {
	h.setCookies()
	return h.writeResponse(r)
}

// redirectXSSI redirects clients by sending a JSON response with an XSSI header.
func (h *Handler) redirectXSSI(r *response.Response) error {
	b, err := r.MarshalXSSI()
	if err != nil {
		h.err(http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return err
	}
	h.setCookies()
	h.w.Header().Set("Content-Type", "application/json")
	_, err = h.w.Write(b)
	return err
}
