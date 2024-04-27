# Commands

## Basics

WSVPN uses a quite simple command system to exchange information between client and server.

They are encoded using JSON and look as follows:

```json
{
    "id":"5e697080-4a99-49da-8dea-2a4513b07c12",
    "command": "...",
    "parameters": {...}
}
```

Each command indicates whether the server or client (or both) can send it. They also have a minimum protocol version (a number exchanged using the `version` command, more on that below). A minimum verison of `0` indicates the command can be sent prior to the version negotiation completing.

## List of commands

### reply

This command is expected to be sent in response to any command that is not itself a `reply` command using the same `id`.

Sample exchange below:

```json
Server command
{
    "id":"3bb1313f-9afd-4e94-8749-4e7da2a9d7ec",
    "command": "set_mtu",
    "parameters": {
        "mtu": 1337
    }
}

Client response
{
    "id":"3bb1313f-9afd-4e94-8749-4e7da2a9d7ec",
    "command": "reply",
    "parameters": {
        "ok": true,
        "message": "OK"
    }
}
```

- Server can send: Yes
- Client can send: Yes
- Minimum protocol version: 0

### version

This command is present to exchange basic capability and version information between the server and client. This must be sent as the first command by both server and client upon connection establishment.

Features are to be considered used / enabled for any feature that both sides include in their `enabled_features` array.

```json
{
    "id":"c1cc3bdb-5e6e-47ec-88fa-1ed360991745",
    "command": "version",
    "parameters": {
        "protocol_version": 12, // Current protocol version
        "version": "wsvpn 1.2.3", // Free-form text of the client/server version
        "enabled_features": [ // Features that are requested
            "fragmentation", // Fragmentation as outlined in PROTOCOL.md
        ]
    }
}
```

- Server can send: Yes
- Client can send: Yes
- Minimum protocol version: 0

### init

This command sends basic information about the tunnel to allow the client to configure its end of the tunnel.

```json
{
    "id": "867dd631-4a25-4894-9297-0324b008e145",
    "command": "init",
    "parameters": {
        "mode": "TUN", // TUN or TAP, the mode of the tunnel
        "do_ip_config": true, // Should the client configure an IP address on the tunnel
        "mtu": 1337, // The MTU on the tunnel, this must always be configured as sent, regardless what "do_ip_config" is set to
        "ip_address": "1.2.3.4/24", // The IP address (with subnet in CIDR format) to configure on the interface if "do_ip_config" is true
        "server_id": "3be18d56-2e98-40ab-b202-c2b4e4b14034", // ID of the server (regenerated on startup currently)
        "client_id": "04513a13-c115-4bf7-bdb0-77ff23df9ba5", // ID of the client (regenerated on connection currently)
    }
}
```

There might be an `enable_fragmentation` boolean present on this packet. This must be ignored for protocol versions `12` and above.

- Server can send: Yes
- Client can send: Yes
- Minimum protocol version: 1

### add_route

Server-issued command to instruct the client to route packets destined for the given subnet over the VPN interface

```json
{
    "id": "736a6d76-bb72-48cc-87cd-5f2e611c0755",
    "command": "add_route",
    "parameters": {
        "route": "1.2.3.0/24"
    }
}
```

- Server can send: Yes
- Client can send: No
- Minimum protocol version: 1

### set_mtu

Server-issued command to tell a client to change the MTU of its interface to the given value

```json
{
    "id": "429196d6-ac96-4a4c-a473-62dd20c4ea55",
    "command": "set_mtu",
    "parameters": {
        "mtu": 1337
    }
}
```

- Server can send: Yes
- Client can send: No
- Minimum protocol version: 7

### message

Exchange free-form messages between client and server

```json
{
    "id": "62a3a600-b187-4813-be57-93c3d7bc7425",
    "command": "message",
    "parameters": {
        "type": "error",
        "message": "Example error"
    }
}
```

- Server can send: Yes
- Client can send: Yes
- Minimum protocol version: 8
