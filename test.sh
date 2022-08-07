#!/bin/sh
set -e

docker build -t wsvpn-test -f Dockerfile.test .
docker run --privileged -v "/dev/net/tun:/dev/net/run" -v "$(pwd):/mnt" -it wsvpn-test
