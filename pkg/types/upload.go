package types

type Upload struct {
	Name     string `yaml:"name,omitempty"`
	WithSudo *bool  `yaml:"with_sudo,omitempty"` // nil implies inherit
	Local    string `yaml:"local"`
	Remote   string `yaml:"remote"`
}

func (u *Upload) GetName() string {
	return u.Name
}
