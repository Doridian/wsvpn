buildfor() {
	export GOOS="$1"
	export GOARCH="$2"
	EXESUFFIX=""
	if [ "$GOOS" == "windows" ]
	then
		EXESUFFIX=".exe"
	fi

	go build -o "dist/client-$GOOS-$GOARCH$EXESUFFIX" ./client
	go build -o "dist/server-$GOOS-$GOARCH$EXESUFFIX" ./server
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
