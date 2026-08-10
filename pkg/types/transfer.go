package types

type Upload struct {
	WithSudo *bool  `yaml:"with_sudo,omitempty"` // nil implies inherit
	Local    string `yaml:"local"`
	Remote   string `yaml:"remote"`
}

type Download struct {
	WithSudo *bool  `yaml:"with_sudo,omitempty"` // nil implies inherit
	Remote   string `yaml:"remote"`
	Local    string `yaml:"local"`
}
