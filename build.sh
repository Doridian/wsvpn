VERSION="$(git describe --tags 2> /dev/null)"
LDFLAGS="-X 'github.com/Doridian/wsvpn/shared.Version=${VERSION}'"

buildfor() {
	export GOOS="$1"
	export GOARCH="$2"
	EXESUFFIX=""
	if [ "$GOOS" == "windows" ]
	then
		EXESUFFIX=".exe"
	fi

	echo "Building for: $GOOS / $GOARCH"

	go build -ldflags="$LDFLAGS" -o "dist/client-$GOOS-$GOARCH$EXESUFFIX" ./client
	go build -ldflags="$LDFLAGS" -o "dist/server-$GOOS-$GOARCH$EXESUFFIX" ./server
}

go mod download

mkdir -p dist

buildfor windows 386
buildfor windows amd64

buildfor linux 386
buildfor linux amd64
buildfor linux arm
buildfor linux arm64
buildfor linux mips
buildfor linux mipsle
buildfor linux mips64
buildfor linux mips64le

buildfor darwin amd64
buildfor darwin arm64
