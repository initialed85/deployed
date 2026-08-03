package types

type Deployment struct {
	Targets []string `yaml:"targets"`
	Steps   []Step   `yaml:"steps"`
}
