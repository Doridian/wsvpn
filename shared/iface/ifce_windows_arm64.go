package iface

import _ "embed"

//go:embed wintun/wintun/bin/arm64/wintun.dll
var wintunDll []byte
