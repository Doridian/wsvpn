#!/bin/sh
set -e

docker build -t wsvpn-testcontainer testcontainer
docker run --rm --sysctl net.ipv6.conf.default.disable_ipv6=0 --cap-add=NET_ADMIN --device /dev/net/tun:/dev/net/tun -v "/dev/net/tun:/dev/net/run" -v "$(pwd):/mnt:ro" -i wsvpn-testcontainer "$@"
