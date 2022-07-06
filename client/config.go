package main

import (
	"io/ioutil"

	"github.com/Doridian/wsvpn/client/clients"
	"github.com/Doridian/wsvpn/shared/cli"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Tunnel struct {
		SetDefaultGateway bool           `yaml:"set-default-gateway"`
		Ping              cli.PingConfig `yaml:"ping"`
	} `yaml:"tunnel"`

	Interface clients.InterfaceConfig `yaml:"interfaces"`

	Scripts struct {
		Up   string `yaml:"up"`
		Down string `yaml:"down"`
	} `yaml:"scripts"`

	Client struct {
		Server   string `yaml:"server"`
		Proxy    string `yaml:"proxy"`
		AuthFile string `yaml:"auth-file"`
		Tls      struct {
			Ca          string        `yaml:"ca"`
			Certificate string        `yaml:"certificate"`
			Key         string        `yaml:"key"`
			Config      cli.TlsConfig `yaml:"config"`
		} `yaml:"tls"`
	}
}

func Load(file string) *Config {
	data, err := ioutil.ReadFile(file)
	if err != nil {
		panic(err)
	}
	out := &Config{}
	err = yaml.Unmarshal(data, out)
	if err != nil {
		panic(err)
	}
	return out
}
