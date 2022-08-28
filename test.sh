#!/bin/sh
set -e

echo "GITHUB_ACTIONS = $GITHUB_ACTIONS"

docker pull ghcr.io/doridian/wsvpn/testcontainer:latest
docker run --rm -e GITHUB_ACTIONS --sysctl net.ipv6.conf.default.disable_ipv6=0 --cap-add=NET_ADMIN --device /dev/net/tun:/dev/net/tun -v "/dev/net/tun:/dev/net/run" -v "$(pwd):/mnt:ro" -i ghcr.io/doridian/wsvpn/testcontainer:latest "$@"
