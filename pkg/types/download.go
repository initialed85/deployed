package types

import (
	"fmt"
)

type Download struct {
	Name     string `yaml:"name,omitempty"`
	WithSudo *bool  `yaml:"with_sudo,omitempty"` // nil implies inherit
	Remote   string `yaml:"remote"`
	Local    string `yaml:"local"`
}

func (d *Download) GetName() string {
	return d.Name
}

func (d *Download) Hash() (string, error) {
	hashes := make([]string, 0)

	//
	// WithSudo
	//

	hash, err := Hash(d.WithSudo)
	if err != nil {
		return "", fmt.Errorf("cannot hash download %#+v because %s", d.Local, err)
	}

	//
	// Remote
	//

	hash, err = Hash(d.Remote)
	if err != nil {
		return "", fmt.Errorf("cannot hash download %#+v because %s", d.Local, err)
	}

	hashes = append(hashes, hash)

	//
	// Local
	//

	hash, err = Hash(d.Local)
	if err != nil {
		return "", fmt.Errorf("cannot hash download %#+v because %s", d.Local, err)
	}

	hashes = append(hashes, hash)

	return Hash(hashes)
}
