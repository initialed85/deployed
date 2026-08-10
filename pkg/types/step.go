package types

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Step struct {
	WithSudo  *bool      `yaml:"with_sudo,omitempty"` // nil implies inherit
	Uploads   []Upload   `yaml:"uploads"`
	Script    string     `yaml:"script"`
	Downloads []Download `yaml:"downloads"`
}

func (s *Step) Validate(rootPaths ...string) error {
	rootPath := ""
	if len(rootPaths) > 0 {
		rootPath = rootPaths[0]
	}

	for i, upload := range s.Uploads {
		_, err := os.Stat(upload.Local)
		if err != nil {
			return fmt.Errorf("script.uploads[%d] refers to local file %#+v that does not exist", i, upload.Local)
		}
	}

	if s.Script != "" && strings.HasPrefix(s.Script, "@") {
		scriptPath := strings.TrimLeft(s.Script, "@")

		if len(rootPath) > 0 {
			scriptPath = filepath.Join(rootPath, scriptPath)
		}

		stat, err := os.Stat(scriptPath)
		if err != nil {
			return fmt.Errorf("script refers to local file %#+v that does not exist", scriptPath)
		}

		if stat.IsDir() {
			return fmt.Errorf("script refers to local file %#+v that is a directory not a file", scriptPath)
		}

		rawScript, err := os.ReadFile(scriptPath)
		if err != nil {
			return fmt.Errorf("script refers to local file %#+v that could not be read because %s", scriptPath, err)
		}

		s.Script = string(rawScript)
	}

	return nil
}
