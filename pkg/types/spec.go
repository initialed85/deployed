package types

import (
	"fmt"
	"strings"
)

type Spec struct {
	Name     string `yaml:"name"`
	WithSudo *bool  `yaml:"with_sudo,omitempty"` // nil implies inherit
	Steps    []Step `yaml:"steps"`
}

func (s *Spec) GetName() string {
	return s.Name
}

func (s *Spec) TestOnlySetName(name string) *Spec {
	s.Name = name

	return s
}

func (s *Spec) Validate(rootPaths ...string) error {
	s.Name = strings.TrimSpace(s.Name)
	if s.Name == "" {
		return fmt.Errorf("%T.name may not be empty", s)
	}

	rootPath := ""
	if len(rootPaths) > 0 {
		rootPath = rootPaths[0]
	}

	for i, step := range s.Steps {
		err := step.Validate(rootPath)
		if err != nil {
			return fmt.Errorf("%T.steps[%d].%s", s, i, err)
		}

		s.Steps[i] = step
	}

	return nil
}
