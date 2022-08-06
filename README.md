# WSVPN

VPN server and client that can talk over WebSocket

## Current features

- WebSocket protocol with and without SSL
- WebTransport protocol (requires SSL as HTTP/3 requires SSL)
- TUN and TAP mode
- Works on Linux, macOS and Windows (Windows requires OpenVPN TAP driver)
- Can authenticate clients via HTTP Basic authentication or mTLS or both

## Download

You can download the latest release binaries at https://github.com/Doridian/wsvpn/releases

Pick the correct binaries for your architecture and OS (`darwin` refers to macOS).

The naming convention is `side-os-architecture` (side being either `client` or `server`)

Some common CPU types and what their architecture is called:
- Intel or AMD CPU: `amd64` on a 64-bit OS, `386` on a 32-bit OS
- Apple Silicon, such as M1: `arm64`
- Raspberry Pi and most other SBCs: `arm64` on a 64-bit OS, `arm32` on a 32-bit OS

For Linux, packed binaries are also offered for very space-constrained devices. These are the ones that end in `-compressed` and are packed using https://github.com/upx/upx

## Building

WSVPN currently requires Golang at least version 1.18 to build successfully. You can use `build.sh` locally if you wish.

Alternatively, the normal go build commands (`go build -o sv ./server`, `go build -o cl ./client`, etc) will work just fine.

## Configuration

You can run the server or client binary with `--print-default-config` and it will give you a commented YAML config file with default options.

Write your customized YAML based on this (you can leave out / remove any option you want to leave at default)

In the below sections, configuration values will be referred to in JavaScript style notation.
As an example, see the following YAML structure:
```yaml
a:
  d:
    e: 1 # <- This is a.d.e
  b:
    c: 0 # <- This is a.b.c
```

*Note:* The server by default is configured to listen on 127.0.0.1:9000 (localhost only) for security reasons.
You can change this to listen externally, but it is only advised to do so if you enabled authentication and TLS.

## Authenticators

### mTLS

#### Server

Set `server.tls.client-ca` on the server, then mTLS will be enabled and required.

If you also enable HTTP Basic authentication, the Common Name (CN) of the certificate presented by the client will have to match the username.

*Note:* This requires TLS to be enabled (`server.tls.key` and `server.tls.certificate` must be set)

#### Client

Set `client.tls.certificate` and `client.tls.key`



### htpasswd

#### Server

Set `server.authenticator.type` to `htpasswd` and `server.authenticator.config` to a htpasswd formatted file.

Such files can be created and managed, for example, by the `htpasswd` CLI tool

#### Client

You can put in credentials using the `scheme://user:password@hostname:port` format in the `client.server` option (such as: `wss://user:pass@example.com:9000`)

Alternatively, set `client.auth-file` to the name of a file with contents of the form `username:password`. This file may contain a blank line at the end, which will be stripped away.


## Limitations

- The server on Windows can currently only work in TAP mode, as the Windows OpenVPN driver does not support dynamic creation of interfaces (which is required as in TUN mode every client gets its own server-side interface)
