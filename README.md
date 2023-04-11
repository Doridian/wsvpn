# WSVPN

[![Go Report Card](https://goreportcard.com/badge/github.com/Doridian/wsvpn)](https://goreportcard.com/report/github.com/Doridian/wsvpn)
[![License: BSD-3-Clause](https://img.shields.io/github/license/Doridian/wsvpn)](https://opensource.org/licenses/BSD-3-Clause)
[![Test](https://github.com/Doridian/wsvpn/actions/workflows/test.yml/badge.svg)](https://github.com/Doridian/wsvpn/actions/workflows/test.yml)
[![Check](https://github.com/Doridian/wsvpn/actions/workflows/check.yml/badge.svg)](https://github.com/Doridian/wsvpn/actions/workflows/check.yml)
[![Release](https://img.shields.io/github/v/release/Doridian/wsvpn)](https://github.com/Doridian/wsvpn/releases)

VPN server and client that can talk over WebSocket or WebTransport

## Potential use cases

- Put VPN server behind reverse proxy, for added security and/or flexibility
- Connect to VPN server from behind very restrictive firewalls as the traffic looks like normal HTTP(S) traffic
- *Very advanced/niche: Connect to the internet from within your browser by writing your own VPN client!*

## Current features

- WebSocket protocol with and without SSL
- WebTransport protocol (requires SSL as HTTP/3 requires SSL)
- TUN and TAP mode
- Works on Linux, macOS and Windows (TAP on Windows requires OpenVPN TAP driver)
- Can authenticate clients via HTTP Basic authentication or mTLS (Mutual TLS) or both

## Download

You can download the latest release binaries at https://github.com/Doridian/wsvpn/releases

Pick the correct binaries for your architecture and OS (`darwin` refers to macOS).

The naming convention is `wsvpn-os-architecture`

Some common CPU types and what their architecture is called:
- Intel or AMD CPU: `amd64` on a 64-bit OS, `386` on a 32-bit OS
- Apple Silicon, such as M1: `arm64`
- Raspberry Pi and most other SBCs: `arm64` on a 64-bit OS, `armv6` on a 32-bit OS

For macOS, universal binaries are offered as `wsvpn-darwin-universal`

## Example configurations

In each of these examples, you run the tunnel as follows:
1. Put the config in a file ending in `.yml`
1. Run the binary with `--config=myfile.yml` and either `--mode=server` or `--mode=client` with the full filename of the file
   1. On Windows, this has to be done in a "Run as Administrator" command prompt, and works like `.\wsvpn-windows-amd64.exe --mode client --config=myfile.yml`
   1. On macOS and linux, this has to be run as root, like: `sudo ./wsvpn-linux-amd64 --mode client --config=myfile.yml`

**Keep in mind that WebTransport should perform better than WebSocket in most scenarios but is considered to be less stable**

[VPN with TLS + htpasswd](https://github.com/Doridian/wsvpn/wiki/Example:-VPN-with-TLS-and-htpasswd-authentication)


*A bit of work might be required to setup an mTLS CA for this one:* [VPN with TLS + mTLS](https://github.com/Doridian/wsvpn/wiki/Example:-VPN-with-TLS-and-mTLS)


## Building

WSVPN currently requires Golang at least version 1.18 to build successfully. You can use `build.py` locally if you wish.

The suggested invocation to build binaries for your local machine would look like: `./build.py --platforms local --architectures local`.

The binaries can be found in the `dist` folder.

## Advanced configurations

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
