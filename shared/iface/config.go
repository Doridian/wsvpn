package iface

type InterfaceConfig struct {
	Name                      string `yaml:"name"`
	Persist                   bool   `yaml:"persist"`
	ComponentID               string `yaml:"component-id"`
	OneInterfacePerConnection bool   `yaml:"one-interface-per-connection"`
}
