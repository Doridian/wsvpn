package iface

import _ "embed"

//go:embed wintun/wintun/bin/386/wintun.dll
var wintunDll []byte
