package cli

import (
	"flag"
)

func LoadFlags(configName string, configHelp string) (*string, *bool) {
	if configHelp == "" {
		configHelp = "Config file name"
	}
	configPtr := flag.String("config", configName, configHelp)
	printDefaultConfigPtr := flag.Bool("print-default-config", false, "Print default config to console")

	flag.Usage = UsageWithVersion

	flag.Parse()

	return configPtr, printDefaultConfigPtr
}
