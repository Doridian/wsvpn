package shared

import (
	"crypto/tls"
	_ "unsafe"
)

func TlsVersionString(version uint16) string {
	switch version {
	case tls.VersionTLS10:
		return "1.0"
	case tls.VersionTLS11:
		return "1.1"
	case tls.VersionTLS12:
		return "1.2"
	case tls.VersionTLS13:
		return "1.3"
	}
	return "Invalid"
}

func TlsVersionNum(version string) uint16 {
	switch version {
	case "1.0":
		return tls.VersionTLS10
	case "1.1":
		return tls.VersionTLS11
	case "1.2":
		return tls.VersionTLS12
	case "1.3":
		return tls.VersionTLS13
	}
	return 0
}

//go:linkname hasAESGCMHardwareSupport crypto/tls.hasAESGCMHardwareSupport
var hasAESGCMHardwareSupport bool

func TlsSetCipherAESPreference(preferAES bool) {
	hasAESGCMHardwareSupport = preferAES
}
