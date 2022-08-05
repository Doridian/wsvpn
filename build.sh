#!/usr/bin/env bash
#set -euo pipefail

VERSION="$(git describe --tags 2> /dev/null)"
if [ -z "$VERSION" ]
then
	VERSION="dev"
fi

LDFLAGS="-X 'github.com/Doridian/wsvpn/shared.Version=${VERSION}'"
DO_DOCKER_PUSH="$1"
DO_DOCKER_TAG_LATEST="$2"

gobuild() {
	MOD="$1"
	go build -ldflags="$LDFLAGS" -o "dist/$MOD-$GOOS-$ARCHNAME$EXESUFFIX" "./$MOD"
}

buildfor() {
	export GOOS="$1"
	export GOARCH="$2"
	export GOARCHSUFFIX="$3"
	EXESUFFIX=""
	if [ "$GOOS" == "windows" ]
	then
		EXESUFFIX=".exe"
	fi

	if [ ! -z "$GOARCHSUFFIX" ]
	then
		GOARCHSUFFIX="-$GOARCHSUFFIX"
	fi

	export ARCHNAME="$GOARCH$GOARCHSUFFIX"
	case "$ARCHNAME" in
		arm)
			if [ ! -z "$GOARM" ]
			then
				export ARCHNAME="arm32v$GOARM"
			fi
			;;
	esac

	echo "Building for: $GOOS / $GOARCH$GOARCHSUFFIX / $ARCHNAME"

	gobuild client
	gobuild server
}

buildmips() {
	export GOMIPS=""
	buildfor "$1" "$2" "$GOMIPS"
	export GOMIPS="softfloat"
	buildfor "$1" "$2" "$GOMIPS"
	export GOMIPS=""
}

buildarm() {
	export GOARM=""
	buildfor "$1" "$2"
	export GOARM="5"
	buildfor "$1" "$2"
	export GOARM="6"
	buildfor "$1" "$2"
	export GOARM="7"
	buildfor "$1" "$2"
	export GOARM=""
}

go mod download

rm -rf dist && mkdir -p dist

buildfor windows 386
buildfor windows amd64
buildfor windows arm64

buildfor linux 386
buildfor linux amd64
buildarm linux arm
buildfor linux arm64
buildmips linux mips
buildmips linux mipsle
buildfor linux mips64
buildfor linux mips64le

buildfor darwin amd64
buildfor darwin arm64

cd dist
sha256sum * > sha256sums.txt
cd ..

dockerbuild() {
	SIDE="$1"
	DOCKERCMD="docker buildx build --platform linux/i386,linux/amd64,linux/arm32/v5,linux/arm32/v6,linux/arm32/v7,linux/arm64"
	DOCKERCMD="$DOCKERCMD -t ghcr.io/doridian/wsvpn/$SIDE:$VERSION"
	if [ ! -z "$DO_DOCKER_TAG_LATEST" ]
	then
		DOCKERCMD="$DOCKERCMD -t ghcr.io/doridian/wsvpn/$SIDE:latest"
	fi
	if [ ! -z "$DO_DOCKER_PUSH" ]
	then
		DOCKERCMD="$DOCKERCMD --push"
	fi
	DOCKERCMD="$DOCKERCMD -f Dockerfile.$SIDE ."

	$DOCKERCMD
}

docker buildx create --use --name multiarch || true
dockerbuild server
dockerbuild client
