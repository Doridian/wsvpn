package iface

import _ "embed" // Required for go:embed_ "embed"

//go:embed wintun/wintun/bin/amd64/wintun.dll
var wintunDll []byte
