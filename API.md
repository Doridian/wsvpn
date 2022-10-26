# API

WSVPN now has a very basic API (for now)

In the server config, you need to set `server.api.enabled` to `true` and (for security reasons) choose which users can access the API via `server.api.users` (case senstive usernames).

## Basics

The API is available on the same port as the VPN server itself on the `/api` path.

### GET /api/clients

Gives a list of all currently connected clients with some info.

```
[
    {
        "client_id": "bfa2980f-2a64-4724-af05-0256c36da9fe",
        "vpn_ip": "192.168.3.2",
        "local_addr": "127.0.0.1:9000",
        "remote_addr": "127.0.0.1:57445"
    }
]
```

### GET /api/clients/{client_id}

Gives the information about a single client, returns 404 status if the client is not connected

```
{
    "client_id": "bfa2980f-2a64-4724-af05-0256c36da9fe",
    "vpn_ip": "192.168.3.2",
    "local_addr": "127.0.0.1:9000",
    "remote_addr": "127.0.0.1:57445"
}
```

### DELETE /api/clients/{client_id}

Disconnects the client (and returns 200 status), returns 404 status if the client is not connected
