// Package runner implements the main SSH-over-WebSocket Relay logic, as defined in
// https://chromium.googlesource.com/apps/libapps/+/HEAD/nassh/doc/relay-protocol.md.
//
// Notes about this implementation:
// * XHR support (/read, /write) is not implemented, clients MUST support secure WebSockets (WSS).
// * Clients MUST support at least TLS 1.2, older versions are not supported.
// * Client validation is not fully implemented; cookies are not signed/encrypted.
package runner
