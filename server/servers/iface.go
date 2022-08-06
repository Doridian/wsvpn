package servers

type InterfaceConfig struct {
	Name        string `yaml:"name"`
	Persist     bool   `yaml:"persist"`
	ComponentId string `yaml:"component-id"`
}
