package types

type Step struct {
	WithSudo  *bool      `yaml:"with_sudo,omitempty"` // nil implies inherit
	Uploads   []Upload   `yaml:"uploads"`
	Script    string     `yaml:"script"`
	Downloads []Download `yaml:"downloads"`
}
