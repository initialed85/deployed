package types

type Pool struct {
	Name    string   `yaml:"name"`
	Targets []string `yaml:"targets"`
}
