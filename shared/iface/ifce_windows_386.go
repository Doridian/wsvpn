package iface

import _ "embed" // Required for go:embed

//go:embed wintun/wintun/bin/x86/wintun.dll
var wintunDll []byte
