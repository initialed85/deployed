package types

type Download struct {
	Name     string `yaml:"name,omitempty"`
	WithSudo *bool  `yaml:"with_sudo,omitempty"` // nil implies inherit
	Remote   string `yaml:"remote"`
	Local    string `yaml:"local"`
}

func (d *Download) GetName() string {
	return d.Name
}
