# SSH-over-WebSocket Relay

[![GoDoc](https://godoc.org/github.com/hazaelsan/ssh-relay?status.svg)](https://godoc.org/github.com/hazaelsan/ssh-relay)

`ssh-relay` is a client/server implementation of Google's SSH-over-WebSocket
Relay, as defined in
https://chromium.googlesource.com/apps/libapps/+/HEAD/nassh/doc/relay-protocol.md.

The primary use case for this is using the [Secure Shell Chrome
App][chrome-app], but the client helper makes it possible to use with ssh's
`ProxyCommand` functionality.

## Features

* Supports client/server version 2 of the Relay Protocol, version 1 is NOT supported.
* Supports WebSockets for the SSH transport, the older XHR-based method is NOT supported.
* Configuration is done almost entirely via [protobuf messages](https://developers.google.com/protocol-buffers/).
* TLS is **required** for all operations, though its options are configurable.

## Building

Building of all components is done via [Bazel](http://bazel.build).

## Components

It's possible to host the Cookie Server and the SSH Relay on the same host, but
in that case you'll need a reverse proxy (e.g., `nginx`) to route requests
accordingly.

### Cookie Server

The Cookie Server is responsible for authenticating/authorizing clients, and
redirecting them to a suitable SSH Relay.

This component is only used for the first phase of the connection, as such it
has minimal requirements and is not latency sensitive.

NOTE: Dynamic SSH Relay selection is not implemented yet, only a single static
SSH Relay is supported.

### SSH Relay

The SSH Relay takes a client that's been authorized by the Cookie Server, and
relays SSH traffic between the server (talking plain SSH) and the client
(talking SSH-over-WebSocket).

The SSH Relay enforces a maximum session lifetime, forcing clients to
re-authenticate against the Cookie Server periodically.

### Helper

This is a helper binary to relay an `ssh(1)` session via the `ProxyCommand`
directive.

#### Example Usage

##### ~/.ssh/config, /etc/ssh/ssh_config

```
# Anything under example.org must go via the WebSocket relay.
Host *.example.org
  ProxyCommand ssh_relay_helper --config=/etc/ssh-relay-helper/config.textpb --host=%h --port=%p
```

##### /etc/ssh-relay-helper/config.textpb
```proto
# DO NOT set host/port in the config proto.

# This can also be passed via --cookie_server_address
# port defaults to 8022 if unspecified.
cookie_server_address: "cookie-server.example.org:8022"

# Settings for talking to the Cookie Server.
cookie_server_transport {
  tls_config {
    cert_file: "/etc/ssh-relay-helper/client.crt"
    key_file: "/etc/ssh-relay-helper/client.key"
    root_ca_certs: "/etc/ssh-relay-helper/ca.crt"
  }
}
```

## Bugs / Feature Requests

All bugs/feature requests should be done via GitHub, pull requests are welcome.

## Disclaimer

**This project is NOT affiliated or endorsed by Google.**

All code was written based on Google's [public documentation][relay-protocol] and minimally tested using the [Secure Shell Chrome App][chrome-app].

[relay-protocol]: https://chromium.googlesource.com/apps/libapps/+/HEAD/nassh/doc/relay-protocol.md
[chrome-app]: https://chrome.google.com/webstore/detail/secure-shell-app/pnhechapfaindjhompbnflcldabbghjo
