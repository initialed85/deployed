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

func (s *Spec) Validate(rootPaths ...string) error {
	rootPath := ""
	if len(rootPaths) > 0 {
		rootPath = rootPaths[0]
	}

	s.Name = strings.TrimSpace(s.Name)
	if s.Name == "" {
		return errors.New("spec has empty name")
	}

	for i, step := range s.Steps {
		err := step.Validate(rootPath)
		if err != nil {
			return fmt.Errorf("spec.steps[%d].%s", i, err)
		}

		s.Steps[i] = step
	}

	return nil
}
