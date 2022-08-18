package iface

import _ "embed"

//go:embed wintun/wintun/bin/amd64/wintun.dll
var wintunDll []byte
