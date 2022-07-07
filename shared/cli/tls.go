package cli

import (
	"crypto/tls"
	"errors"
	"strings"

	"github.com/Doridian/wsvpn/shared"
)

type TlsConfig struct {
	MinVersion       string `yaml:"min-version"`
	MaxVersion       string `yaml:"max-version"`
	Insecure         bool   `yaml:"insecure"`
	CipherPreference string `yaml:"cipher-preference"`
}

func TlsUseConfig(tlsConfig *tls.Config, fileConfig *TlsConfig) {
	tlsConfig.MinVersion = shared.TlsVersionNum(fileConfig.MinVersion)
	tlsConfig.MaxVersion = shared.TlsVersionNum(fileConfig.MaxVersion)

	switch strings.ToUpper(fileConfig.CipherPreference) {
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
