package clients

type InterfaceConfig struct {
	Tap struct {
		Name        string `yaml:"name"`
		Persist     bool   `yaml:"persist"`
		ComponentId string `yaml:"component-id"`
	} `yaml:"tap"`
	Tun struct {
		Name        string `yaml:"name"`
		ComponentId string `yaml:"component-id"`
	} `yaml:"tun"`
}
