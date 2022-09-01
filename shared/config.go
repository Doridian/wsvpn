package shared

import (
	"io"
	"os"

	"gopkg.in/yaml.v3"
)

func LoadConfigFile(file string, out interface{}) error {
	fh, err := os.Open(file) // #nosec G304 -- Variable provided only from internal / CLI sources
	if err != nil {
		return err
	}
	defer fh.Close() // #nosec G307 -- Closing a file with defer is not unsafe

	return LoadConfigReader(fh, out)
}

func LoadConfigReader(reader io.Reader, out interface{}) error {
	decoder := yaml.NewDecoder(reader)
	decoder.KnownFields(true)
	return decoder.Decode(out)
}
