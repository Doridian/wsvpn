package shared

import (
	"os"

	"gopkg.in/yaml.v3"
)

func LoadConfig(file string, out interface{}) error {
	fh, err := os.Open(file)
	if err != nil {
		return err
	}
	defer fh.Close()

	decoder := yaml.NewDecoder(fh)
	decoder.KnownFields(true)
	return decoder.Decode(out)
}
