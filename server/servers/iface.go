package servers

type InterfacesConfig struct {
	Tap struct {
		Name        string `yaml:"name"`
		Persist     bool   `yaml:"persist"`
		ComponentId string `yaml:"component-id"`
	} `yaml:"tap"`
	Tun struct {
		NamePrefix string `yaml:"name-prefix"`
	} `yaml:"tun"`
}
