package servers

type InterfacesConfig struct {
	Name        string `yaml:"name"`
	Persist     bool   `yaml:"persist"`
	ComponentId string `yaml:"component-id"`
}
