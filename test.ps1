#!/usr/bin/env pwsh

docker build -t wsvpn-testcontainer testcontainer
docker run --rm --cap-add=NET_ADMIN --cap-add=NET_RAW --device /dev/net/tun:/dev/net/tun -v "/dev/net/tun:/dev/net/run" -v "${pwd}:/mnt:ro" -i wsvpn-testcontainer $args
