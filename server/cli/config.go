package cli

import (
	_ "embed" // Required for go:embed
	"log"
	"strings"

	"github.com/Doridian/wsvpn/shared"
	shared_cli "github.com/Doridian/wsvpn/shared/cli"
	"github.com/Doridian/wsvpn/shared/features"
	"github.com/Doridian/wsvpn/shared/iface"
)

//go:embed server.example.yml
var defaultConfig string

type Config struct {
	Tunnel struct {
		MTU                      int             `yaml:"mtu"`
		Subnet                   string          `yaml:"subnet"`
		Mode                     string          `yaml:"mode"`
		AllowClientToClient      bool            `yaml:"allow-client-to-client"`
		AllowIPSpoofing          bool            `yaml:"allow-ip-spoofing"`
		AllowUnknownEtherTypes   bool            `yaml:"allow-unknown-ether-types"`
		AllowMACChanging         bool            `yaml:"allow-mac-changing"`
		AllowedMACsPerConnection int             `yaml:"allowed-macs-per-connection"`
		Features                 features.Config `yaml:"features"`
		IPConfig                 struct {
			Local  bool `yaml:"local"`
			Remote bool `yaml:"remote"`
		} `yaml:"ip-config"`
		Ping shared_cli.PingConfig `yaml:"ping"`
	} `yaml:"tunnel"`

	Interface iface.InterfaceConfig `yaml:"interface"`

	Scripts shared.EventConfig `yaml:"scripts"`

	Server struct {
		Listen      string `yaml:"listen"`
		EnableHTTP3 bool   `yaml:"enable-http3"`
		TLS         struct {
			ClientCA    string               `yaml:"client-ca"`
			Certificate string               `yaml:"certificate"`
			Key         string               `yaml:"key"`
			Config      shared_cli.TLSConfig `yaml:"config"`
		} `yaml:"tls"`
		Authenticator struct {
			Type   string `yaml:"type"`
			Config string `yaml:"config"`
		} `yaml:"authenticator"`
		MaxConnectionsPerUser     int    `yaml:"max-connections-per-user"`
		MaxConnectionsPerUserMode string `yaml:"max-connections-per-user-mode"`
		WebsiteDirectory          string `yaml:"website-directory"`
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
