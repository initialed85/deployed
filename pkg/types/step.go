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
	ID        uuid.UUID  `yaml:"-"`
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

	//
	// make sure any upload files or folders exist + apply the workspace path
	//

	for i, upload := range s.Uploads {
		if strings.HasPrefix(upload.Local, "?") {
			continue
		}

		if len(rootPath) > 0 && !strings.HasPrefix(upload.Local, "/") {
			upload.Local = filepath.Join(rootPath, upload.Local)
		}

		_, err := os.Stat(upload.Local)
		if err != nil {
			return fmt.Errorf("script.uploads[%d] refers to local file or folder %#+v that does not exist", i, upload.Local)
		}

		s.Uploads[i] = upload
	}

	//
	// resolve any external script files
	//

	if s.Script != "" && strings.HasPrefix(s.Script, "@") {
		scriptPath := strings.TrimLeft(s.Script, "@")

		if len(rootPath) > 0 && !strings.HasPrefix(scriptPath, "/") {
			scriptPath = filepath.Join(rootPath, scriptPath)
		}

		stat, err := os.Stat(scriptPath)
		if err != nil {
			return fmt.Errorf("script refers to local file or folder %#+v that does not exist", scriptPath)
		}

		if stat.IsDir() {
			return fmt.Errorf("script refers to local file or folder %#+v that is a directory not a file", scriptPath)
		}

		rawScript, err := os.ReadFile(scriptPath)
		if err != nil {
			return fmt.Errorf("script refers to local file or folder %#+v that could not be read because %s", scriptPath, err)
		}

		s.Script = string(rawScript)
	}

	//
	// apply the workspace path to the downloads
	//

	for i, download := range s.Downloads {
		if len(rootPath) > 0 && !strings.HasPrefix(download.Local, "/") {
			download.Local = filepath.Join(rootPath, download.Local)
		}

		s.Downloads[i] = download
	}

	return nil
}

func (s *Step) GetID() uuid.UUID {
	if s.ID == zeroUUID {
		s.ID = uuid.Must(uuid.NewRandom())
	}

	return s.ID
}

func (s *Step) Hash() (string, error) {
	hashes := make([]any, 0)

	//
	// WithSudo
	//

	err := func() error {
		hash, err := Hash(s.WithSudo)
		if err != nil {
			return err
		}

		hashes = append(hashes, hash)

		return nil
	}()
	if err != nil {
		return "", err
	}

	//
	// Uploads
	//

	err = func() error {
		for _, upload := range s.Uploads {
			hash, err := upload.Hash()
			if err != nil {
				return err
			}

			hashes = append(hashes, hash)
		}
		return nil
	}()
	if err != nil {
		return "", err
	}

	//
	// Script
	//

	err = func() error {
		hash, err := Hash(s.Script)
		if err != nil {
			return err
		}

		hashes = append(hashes, hash)

		return nil
	}()
	if err != nil {
		return "", err
	}

	//
	// Downloads
	//

	err = func() error {
		for _, download := range s.Downloads {
			hash, err := download.Hash()
			if err != nil {
				return err
			}

			hashes = append(hashes, hash)
		}

		return nil
	}()
	if err != nil {
		return "", err
	}

	return Hash(hashes)
}

func (s *Step) GetHashFileName(specs ...*Spec) string {
	var spec *Spec
	if len(specs) > 0 {
		spec = specs[0]
	}

	return GetHashFileName("step", spec)
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
