# WSVPN protocol(s)

WSVPN's protocol has two big layers. The transport (WebTransport, WebSocket) specific layer, and the universal layer.

## Universal layer

The universal layer is comprised of three big parts.

### Ping/pong

These are used to ensure the connection is still intact, even if there is no traffic currently being tunneled. They consist of a PING packet as a request and a PONG packet as a response.

### Control/command

These are used to exchange information, such as IPs/subnets, MTU, versions, ...

Please see [COMMANDS.md](COMMANDS.md) for a complete description of all current commands and how the protocol's request/response works.

The minimal set of commands necessary to establish a tunnel are `reply`, `version` and `init`.

### Data packets

These contain all tunneled traffic. These look different depending if fragmentation is enabled (the default) or not.

Fragmentation must be considered enabled under **any** of the following conditions:

1. The protocol version of both sides is `>= 12` and `enabled_features` of the `version` packet **of both sides** includes `fragmentation`
1. The protocol version of either side is `= 10` and the other side is `>= 10`
1. The protocol version of either side is `= 11` and the other side is `>= 11` and the server sends `enable_fragmentation` as `true` in the `init` command

If fragmentation is **disabled**, each data packet contains exactly one tunneled packet (such as an Ethernet or IP frame)

If fragmentation is **enabled**, then the packets look as follows:

**If the packet fits in a single fragment:** The packet is prefixed by a single byte of 0b10000000 (Highest bit set, all other bits 0)

**If the packet does not fit in a single fragment:** The packet is prefixed by 5 bytes as follows:

The first byte's highest bit indicates if this is the last fragment. The 7 remaining bits indicate the fragment index, which is the index of the fragment within a single packet.

The following 4 bytes indicate the packet index, which is for the entire packet to allow re-assembling the fragments of the same packet back together if multiple packets are in-flight at once.

## Transport specific layer

### WebSocket (and secure WebSocket)

1. Ping/pong packets use the native ping/pong packets of WebSocket

1. Control/command packets use WebSocket text/utf-8 messages

1. Data packets use WebSocket binary messages

### WebTransport

WebTransport establishes a single bi-directional stream and enables datagram support

1. Ping/pong packets send a single byte over the stream (`1` for ping, `2` for pong)

1. Control/command packets also uses the stream. It sends a byte of `0` first, followed by two bytes forming an unsigned 16-bit integer (MSB-first) indicating the length of the following message, and then the message itself

1. Data packets use WebTransport datagrams (the stream ID used for these depends on the enablement of the `datagram_id_0` feature outline in the `version` command in [COMMANDS.md](COMMANDS.md))
