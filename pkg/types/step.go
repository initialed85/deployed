package types

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/initialed85/deployed/pkg/connection/connection_types"
)

type Step struct {
	ID        uuid.UUID  `yaml:"-" json:"-"`
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
		if strings.HasPrefix(upload.Local, "?") {
			continue
		}

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

func (s *Step) GetID() uuid.UUID {
	return s.ID
}

func (s *Step) Hash() (string, error) {
	return Hash(s)
}

func (s *Step) HashFile(specs ...*Spec) string {
	var spec *Spec
	if len(specs) > 0 {
		spec = specs[0]
	}

	return HashFile("step", spec)
}

func (s *Step) WriteHashLocally(spec *Spec) (string, func(), error) {
	return WriteHashLocally(s, spec)
}

func (s *Step) LocalAndRemoteHashesMatch(c connection_types.Deployable, spec *Spec) (bool, error) {
	return LocalAndRemoteHashesMatch("step", s, c, spec)
}

func (s *Step) WriteAttemptedHashToRemote(c connection_types.Deployable, spec *Spec) error {
	return WriteAttemptedHashToRemote(s, c, spec)
}

func (s *Step) CommitRemoteHash(c connection_types.Deployable, spec *Spec) error {
	return CommitRemoteHash(s, c, spec)
}
