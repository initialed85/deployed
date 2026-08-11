package types

import (
	"errors"
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
		return errors.New("spec.name may not be empty")
	}

	for i, step := range s.Steps {
		err := step.Validate(rootPaths...)
		if err != nil {
			return fmt.Errorf("spec.steps[%d].%s", i, err)
		}

		s.Steps[i] = step
	}

	return nil
}
