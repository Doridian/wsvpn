package main

import (
	_ "embed"
	"log"
	"strings"
	"time"

	"github.com/Doridian/wsvpn/client/clients"
	"github.com/Doridian/wsvpn/shared"
	"github.com/Doridian/wsvpn/shared/cli"
)

//go:embed client.example.yml
var defaultConfig string

type Config struct {
	Tunnel struct {
		SetDefaultGateway bool           `yaml:"set-default-gateway"`
		Ping              cli.PingConfig `yaml:"ping"`
	} `yaml:"tunnel"`

	Interface clients.InterfaceConfig `yaml:"interface"`

	Scripts struct {
		Up   string `yaml:"up"`
		Down string `yaml:"down"`
	} `yaml:"scripts"`

	Client struct {
		Server             string        `yaml:"server"`
		Proxy              string        `yaml:"proxy"`
		AuthFile           string        `yaml:"auth-file"`
		AutoReconnectDelay time.Duration `yaml:"auto-reconnect-delay"`
		Tls                struct {
			Ca          string        `yaml:"ca"`
			Certificate string        `yaml:"certificate"`
			Key         string        `yaml:"key"`
			ServerName  string        `yaml:"server-name"`
			Config      cli.TlsConfig `yaml:"config"`
		} `yaml:"tls"`
	}
}

func Load(file string) *Config {
	out := &Config{}

	err := shared.LoadConfigReader(strings.NewReader(defaultConfig), out)
	if err != nil {
		log.Printf("ERROR LOADING DEFAULT CONFIG. THIS SHOULD NEVER HAPPEN!")
		panic(err)
	}

	err = shared.LoadConfigFile(file, out)
	if err != nil {
		panic(err)
	}

	return out
}

func GetDefaultConfig() string {
	return defaultConfig
}
