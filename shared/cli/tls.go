package cli

import (
	"crypto/tls"
	"log"
	"os"

	"github.com/Doridian/wsvpn/shared"
)

type TLSConfig struct {
	MinVersion string `yaml:"min-version"`
	MaxVersion string `yaml:"max-version"`
	KeyLogFile string `yaml:"key-log-file"`
	Insecure   bool   `yaml:"insecure"`
}

func TLSUseConfig(tlsConfig *tls.Config, fileConfig *TLSConfig) error {
	tlsConfig.MinVersion = shared.TLSVersionNum(fileConfig.MinVersion)
	tlsConfig.MaxVersion = shared.TLSVersionNum(fileConfig.MaxVersion)

	if fileConfig.KeyLogFile != "" {
		fh, err := os.OpenFile(fileConfig.KeyLogFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
		if err != nil {
			return err
		}
		log.Print("WARNING!!! TLS secret key logging is enabled. This can cause severe security issues! Make sure this is what you want!")
		tlsConfig.KeyLogWriter = fh
	}

	return nil
}
