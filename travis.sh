buildfor() {
	export GOOS="$1"
	export GOARCH="$2"
	export GIMME_OS="$GOOS"
	export GIMME_ARCH="$GOARCH"
	EXESUFFIX=""
	if [ "$GOOS" == "windows" ]
	then
		EXESUFFIX=".exe"
	fi

	go get -d -v ./...

	go build -o "$HOME/binaries/client-$GOOS-$GOARCH$EXESUFFIX" github.com/Doridian/wsvpn/client
	go build -o "$HOME/binaries/server-$GOOS-$GOARCH$EXESUFFIX" github.com/Doridian/wsvpn/server
}
