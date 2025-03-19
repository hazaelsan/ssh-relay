// Package runner implements the main Cookie Server logic, as defined in
// https://chromium.googlesource.com/apps/libapps/+/HEAD/nassh/docs/relay-protocol.md.
//
// Notes about this implementation:
// * THE CURRENT IMPLEMENTATION IS ONLY A PROOF OF CONCEPT. It offers no additional security over a regular SSH session yet.
// * Clients MUST support at least TLS 1.2, older versions are not supported.
// * Client validation is not implemented, and cookies are not signed/encrypted.
// * Dynamic SSH Relay selection is not implemented yet.
//
// A typical request looks like
// http://HOST:8022/cookie?ext=foo&path=html/nassh_google_relay.html&version=2&method=js-redirect
//
// After authnz has completed, the server will send a redirect to
// chrome-extension://extid/html/nassh_google_relay.html#USER@RELAY_HOST
package runner
