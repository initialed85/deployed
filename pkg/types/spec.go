package types

type Spec struct {
	Name     string `yaml:"name"`
	WithSudo *bool  `yaml:"with_sudo,omitempty"` // nil implies inherit
	Steps    []Step `yaml:"steps"`
}
