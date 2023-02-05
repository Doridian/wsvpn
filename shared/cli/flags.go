package cli

import (
	"flag"
	"log"
	"os"
	"runtime/pprof"
)

func LoadFlags(configName string, configHelp string) (*string, *bool) {
	if configHelp == "" {
		configHelp = "Config file name"
	}
	configPtr := flag.String("config", configName, configHelp)
	printDefaultConfigPtr := flag.Bool("print-default-config", false, "Print default config to console")
	cpuProfPtr := flag.String("cpuprofile", "", "CPU profile output file")

	flag.Usage = UsageWithVersion

	flag.Parse()

	cpuProf := *cpuProfPtr
	if cpuProf != "" {
		f, err := os.Create(cpuProf)
		if err != nil {
			log.Fatal(err)
		}
		pprof.StartCPUProfile(f)
	}

	return configPtr, printDefaultConfigPtr
}
