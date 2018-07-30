buildfor() {
	GOOS="$1"
	GOARCH="$2"
	EXESUFFIX=""
	if [ "$GOOS" == "windows" ]
	then
		EXESUFFIX=".exe"
	fi

	go build -o "$HOME/binaries/client-$GOOS-$GOARCH$EXESUFFIX" -v client
	go build -o "$HOME/binaries/server-$GOOS-$GOARCH$EXESUFFIX" -v server
}
