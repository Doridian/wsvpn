#!/usr/bin/env pwsh

docker pull ghcr.io/doridian/wsvpn/testcontainer:latest
docker run -e GITHUB_ACTIONS --cap-add=NET_ADMIN --device /dev/net/tun:/dev/net/tun -v "/dev/net/tun:/dev/net/run" -v "${pwd}:/mnt:ro" -i ghcr.io/doridian/wsvpn/testcontainer:latest $args
