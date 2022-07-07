package shared

import (
	"io"
	"os"

	"gopkg.in/yaml.v3"
)

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
