package types

import (
	"fmt"

	"github.com/google/uuid"
)

var zeroUUID = uuid.UUID{}
var zeroUUIDString = zeroUUID.String()

type Deployment struct {
	ID       uuid.UUID `yaml:"id"`
	WithSudo *bool     `yaml:"with_sudo,omitempty"` // non-nil will override
	Spec     Spec      `yaml:"spec"`
	Target   string    `yaml:"target"`
}

func (d *Deployment) Validate() {
	if d.ID.String() == zeroUUIDString {
		d.ID = uuid.Must(uuid.NewRandom())
	}
}

func (d *Deployment) Hash() (string, error) {
	if d.ID.String() == zeroUUIDString {
		return "", fmt.Errorf("assertion failed: %#+v has no ID set; have you called .Validate()?", d)
	}

	c := Deployment{
		ID:       zeroUUID,
		WithSudo: d.WithSudo,
		Spec:     d.Spec,
		Target:   d.Target,
	}

	return Hash(c)
}

func (d *Deployment) HashFile() string {
	return fmt.Sprintf("deployed-%s-hash.txt", d.Spec.Name)
}
