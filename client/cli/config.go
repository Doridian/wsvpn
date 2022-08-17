package cli

import (
	_ "embed"
	"log"
	"strings"
	"time"

	"github.com/Doridian/wsvpn/shared"
	shared_cli "github.com/Doridian/wsvpn/shared/cli"
	"github.com/Doridian/wsvpn/shared/features"
)

//go:embed client.example.yml
var defaultConfig string

type Config struct {
	Tunnel struct {
		SetDefaultGateway bool                    `yaml:"set-default-gateway"`
		Ping              shared_cli.PingConfig   `yaml:"ping"`
		Features          features.FeaturesConfig `yaml:"features"`
	} `yaml:"tunnel"`

	Interface shared.InterfaceConfig `yaml:"interface"`

	Scripts shared.EventConfig `yaml:"scripts"`

	Client struct {
		Server             string        `yaml:"server"`
		Proxy              string        `yaml:"proxy"`
		AuthFile           string        `yaml:"auth-file"`
		AutoReconnectDelay time.Duration `yaml:"auto-reconnect-delay"`
		Tls                struct {
			Ca          string               `yaml:"ca"`
			Certificate string               `yaml:"certificate"`
			Key         string               `yaml:"key"`
			ServerName  string               `yaml:"server-name"`
			Config      shared_cli.TlsConfig `yaml:"config"`
		} `yaml:"tls"`
	}
}

func Load(file string) (*Config, error) {
	out := &Config{}

	err := shared.LoadConfigReader(strings.NewReader(defaultConfig), out)
	if err != nil {
		log.Printf("ERROR LOADING DEFAULT CONFIG. THIS SHOULD NEVER HAPPEN!")
		return nil, err
	}

	err = shared.LoadConfigFile(file, out)
	if err != nil {
		return nil, err
	}

	return out, nil
}

func GetDefaultConfig() string {
	return defaultConfig
}
