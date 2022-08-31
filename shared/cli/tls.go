package cli

import (
	"crypto/tls"
	"errors"
	"log"
	"os"
	"strings"

	"github.com/Doridian/wsvpn/shared"
)

type TlsConfig struct {
	MinVersion       string `yaml:"min-version"`
	MaxVersion       string `yaml:"max-version"`
	KeyLogFile       string `yaml:"key-log-file"`
	Insecure         bool   `yaml:"insecure"`
	CipherPreference string `yaml:"cipher-preference"`
}

func TlsUseConfig(tlsConfig *tls.Config, fileConfig *TlsConfig) error {
	tlsConfig.MinVersion = shared.TlsVersionNum(fileConfig.MinVersion)
	tlsConfig.MaxVersion = shared.TlsVersionNum(fileConfig.MaxVersion)

	if fileConfig.KeyLogFile != "" {
		fh, err := os.OpenFile(fileConfig.KeyLogFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
		if err != nil {
			return err
		}
		log.Print("WARNING!!! TLS secret key logging is enabled. This can cause severe security issues! Make sure this is what you want!")
		tlsConfig.KeyLogWriter = fh
	}

	switch strings.ToUpper(fileConfig.CipherPreference) {
	case "AES":
		shared.TlsSetCipherAESPreference(true)
	case "CHACHA":
		shared.TlsSetCipherAESPreference(false)
	case "":
		break
	default:
		return errors.New("invalid TLS preference. Must be blank, AES or CHACHA")
	}

	return nil
}
