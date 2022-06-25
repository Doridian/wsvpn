package cli

import (
	"crypto/tls"
	"errors"
	"flag"
	"strings"

	"github.com/Doridian/wsvpn/shared"
)

var tlsMinVersion = flag.String("tls-min-version", "1.2", "Minimum TLS version")
var tlsMaxVersion = flag.String("tls-max-version", "1.3", "Maximum TLS version")
var tlsCipherPreference = flag.String("tls-cipher-preference", "", "Prefer AES ciphers (AES), or ChaCha ciphers (CHACHA), don't specify for default behaviour")

func TlsUseFlags(tlsConfig *tls.Config) {
	tlsConfig.MinVersion = shared.TlsVersionNum(*tlsMinVersion)
	tlsConfig.MaxVersion = shared.TlsVersionNum(*tlsMaxVersion)

	switch strings.ToUpper(*tlsCipherPreference) {
	case "AES":
		shared.TlsSetCipherAESPreference(true)
	case "CHACHA":
		shared.TlsSetCipherAESPreference(false)
	case "":
		break
	default:
		panic(errors.New("invalid TLS preference. Must be blank, AES or CHACHA"))
	}
}
