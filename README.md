# SSH-over-WebSocket Relay

[![GoDoc](https://godoc.org/github.com/hazaelsan/ssh-relay?status.svg)](https://godoc.org/github.com/hazaelsan/ssh-relay)

`ssh-relay` is a client/server implementation of Google's SSH-over-WebSocket
Relay, as defined in
https://chromium.googlesource.com/apps/libapps/+/HEAD/nassh/docs/relay-protocol.md.

The primary use case for this is using the [Secure Shell Chrome
App][chrome-app] and the [Secure Shell Chrome Extension][chrome-extension], but
the client helper makes it possible to use with ssh's `ProxyCommand`
functionality.

## Features

* **NEW:** Supports the `corp-relay-v4@google.com` version of the Relay Protocol.
* Supports client/server version 2 of the Cookie Protocol.
  * Version 1 is supported by the Cookie Server, though this version is deprecated.
* Supports WebSockets for the SSH transport (via `/connect`).
  * The older XHR-based method (via `/read` and `/write`) is NOT supported.
* Configuration is done almost entirely via [protobuf messages](https://developers.google.com/protocol-buffers/).
* TLS is **required** for all operations, though its options are configurable.

## Building

Building of all components is done via [Bazel](http://bazel.build).

## Chrome Extension / App Usage

Add something like this to the `SSH relay server options`:

### `corp-relay-v4@google.com` (New Protocol)
```none
--proxy-host=cookie-server.example.org --proxy-port=8022 --use-ssl --proxy-mode=corp-relay-v4@google.com
```

### `corp-relay@google.com` (Old Protocol)
```none
--relay-protocol=v2 --report-ack-latency=true --report-connect-attempts=true --proxy-host=cookie-server.example.org --proxy-port=8022 --use-ssl
```

If you are using a security key (e.g., the [Titan Security
Key](https://store.google.com/us/product/titan_security_key)), add
`--ssh-agent=gnubby` to the options above.

## Components

NOTE: It's possible to host the Cookie Server and the SSH Relay on the same
host, but in that case you'll need a reverse proxy (e.g., `nginx`) to route
requests accordingly.

### Cookie Server

The Cookie Server is responsible for authenticating/authorizing clients, and
redirecting them to a suitable SSH Relay.

This component is only used for the first phase of the connection, as such it
has minimal requirements and is not latency sensitive.

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
  ProxyCommand ssh_relay_helper --config=/etc/ssh-relay-helper/config.textproto --host=%h --port=%p
```

##### /etc/ssh-relay-helper/config.textproto
```proto
# DO NOT set host/port in the config proto.

# This can also be passed via --cookie_server_address,
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

See the [examples
directory](https://github.com/hazaelsan/ssh-relay/tree/master/examples) for
additional configuration examples.

## Bugs / Feature Requests

All bugs/feature requests should be done via GitHub, pull requests are welcome.

## Disclaimer

**This project is NOT affiliated or endorsed by Google.**

All code was written based on Google's [public documentation][relay-protocol]
and minimally tested using the [Secure Shell Chrome App][chrome-app] and the
[Secure Shell Chrome Extension][chrome-extension].

[relay-protocol]: https://chromium.googlesource.com/apps/libapps/+/HEAD/nassh/docs/relay-protocol.md
[chrome-app]: https://chrome.google.com/webstore/detail/secure-shell-app/pnhechapfaindjhompbnflcldabbghjo
[chrome-extension]: https://chrome.google.com/webstore/detail/secure-shell/iodihamcpbpeioajjeobimgagajmlibd
