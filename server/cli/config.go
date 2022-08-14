package cli

import (
	_ "embed"
	"log"
	"strings"

	"github.com/Doridian/wsvpn/shared"
	shared_cli "github.com/Doridian/wsvpn/shared/cli"
	"github.com/Doridian/wsvpn/shared/features"
)

//go:embed server.example.yml
var defaultConfig string

type Config struct {
	Tunnel struct {
		Mtu                      int                     `yaml:"mtu"`
		Subnet                   string                  `yaml:"subnet"`
		Mode                     string                  `yaml:"mode"`
		AllowClientToClient      bool                    `yaml:"allow-client-to-client"`
		AllowIpSpoofing          bool                    `yaml:"allow-ip-spoofing"`
		AllowUnknownEtherTypes   bool                    `yaml:"allow-unknown-ether-types"`
		AllowMacChanging         bool                    `yaml:"allow-mac-changing"`
		AllowedMacsPerConnection int                     `yaml:"allowed-macs-per-connection"`
		Features                 features.FeaturesConfig `yaml:"features"`
		IpConfig                 struct {
			Local  bool `yaml:"local"`
			Remote bool `yaml:"remote"`
		} `yaml:"ip-config"`
		Ping shared_cli.PingConfig `yaml:"ping"`
	} `yaml:"tunnel"`

	Interface shared.InterfaceConfig `yaml:"interface"`

	Scripts shared.EventConfig `yaml:"scripts"`

	Server struct {
		Listen      string `yaml:"listen"`
		EnableHTTP3 bool   `yaml:"enable-http3"`
		Tls         struct {
			ClientCa    string               `yaml:"client-ca"`
			Certificate string               `yaml:"certificate"`
			Key         string               `yaml:"key"`
			Config      shared_cli.TlsConfig `yaml:"config"`
		} `yaml:"tls"`
		Authenticator struct {
			Type   string `yaml:"type"`
			Config string `yaml:"config"`
		} `yaml:"authenticator"`
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
