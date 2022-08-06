package shared

import (
	"io"
	"os"

	"gopkg.in/yaml.v3"
)

type InterfaceConfig struct {
	Name                      string `yaml:"name"`
	Persist                   bool   `yaml:"persist"`
	ComponentId               string `yaml:"component-id"`
	OneInterfacePerConnection bool   `yaml:"one-interface-per-connection"`
}

func LoadConfigFile(file string, out interface{}) error {
	fh, err := os.Open(file)
	if err != nil {
		return err
	}
	defer fh.Close()

	return LoadConfigReader(fh, out)
}

func LoadConfigReader(reader io.Reader, out interface{}) error {
	decoder := yaml.NewDecoder(reader)
	decoder.KnownFields(true)
	return decoder.Decode(out)
}
