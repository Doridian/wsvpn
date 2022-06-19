#!/usr/bin/env bash
#set -euo pipefail

VERSION="$(git describe --tags 2> /dev/null)"
LDFLAGS="-X 'github.com/Doridian/wsvpn/shared.Version=${VERSION}'"

gobuild() {
	MOD="$1"
	go build -ldflags="$LDFLAGS" -o "dist/$MOD-$GOOS-$GOARCH$GOARCHSUFFIX$EXESUFFIX" "./$MOD"
}

buildfor() {
	export GOOS="$1"
	export GOARCH="$2"
	EXESUFFIX=""
	GOARCHSUFFIX=""
	if [ "$GOOS" == "windows" ]
	then
		EXESUFFIX=".exe"
	fi

	case "$GOARCH"
	in
		mips|mipsle)
			GOARCHSUFFIX="$GOMIPS"
			;;
		arm)
			GOARCHSUFFIX="$GOARM"
			;;
	esac

	if [ ! -z "$GOARCHSUFFIX" ]
	then
		GOARCHSUFFIX="-$GOARCHSUFFIX"
	fi

	echo "Building for: $GOOS / $GOARCH$GOARCHSUFFIX"

	gobuild client
	gobuild server
}

buildmips() {
	export GOMIPS=""
	buildfor "$1" "$2"
	export GOMIPS="softfloat"
	buildfor "$1" "$2"
	export GOMIPS=""
}

buildarm() {
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
