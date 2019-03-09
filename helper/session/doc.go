// Package session implements an SSH-over-WebSocket Relay client session.
//
// Sessions are established in two parts:
// * /proxy: Tells the Relay to set up the SSH connection, returns a Session ID
// * /connect: SSH-over-WebSocket Relay session
//
// NOTE: Reconnections are not implemented.
package session
