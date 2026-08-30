package types

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

type Upload struct {
	Name     string `yaml:"name,omitempty"`
	WithSudo *bool  `yaml:"with_sudo,omitempty"` // nil implies inherit
	Local    string `yaml:"local"`
	Remote   string `yaml:"remote"`
}

func (u *Upload) GetName() string {
	return u.Name
}

func (u *Upload) Hash() (string, error) {
	hash, err := func() (string, error) {
		if strings.HasPrefix(u.Local, "?") {
			return "", fmt.Errorf("local %#+v not yet resolved to an actual file", u.Local)
		}

		stat, err := os.Stat(u.Local)
		if err != nil {
			return "", err
		}

		hashes := make([]string, 0)

		//
		// WithSudo
		//

		func() error {
			hash, err := Hash(u.WithSudo)
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
		// Local
		//

		func() error {
			if !stat.IsDir() {
				f, err := os.Open(u.Local)
				if err != nil {
					return err
				}

				hash, err := HashFile(f)
				if err != nil {
					return err
				}

				hashes = append(hashes, hash)

				return nil
			}

			err = filepath.Walk(u.Local, func(path string, info fs.FileInfo, err error) error {
				if err != nil {
					return err
				}

				if info.IsDir() {
					return nil
				}

				f, err := os.Open(path)
				if err != nil {
					return err
				}

				hash, err := HashFile(f)
				if err != nil {
					return err
				}

				hashes = append(hashes, hash)

				return nil
			})
			if err != nil {
				return err
			}

			return nil
		}()
		if err != nil {
			return "", err
		}

		//
		// Remote
		//

		func() error {
			hash, err := Hash(u.Remote)
			if err != nil {
				return err
			}

			hashes = append(hashes, hash)

			return nil
		}()
		if err != nil {
			return "", err
		}

		return Hash(hashes)
	}()
	if err != nil {
		return "", fmt.Errorf("cannot hash upload %#+v because %s", u.Local, err)
	}

	return hash, nil
}
