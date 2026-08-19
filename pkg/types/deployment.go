package types

import (
	"github.com/google/uuid"
	"github.com/initialed85/deployed/pkg/connection/connection_types"
)

var zeroUUID = uuid.UUID{}
var zeroUUIDString = zeroUUID.String()

type Deployment struct {
	ID            uuid.UUID `yaml:"-" json:"-"`
	ForceWithSudo *bool     `yaml:"force_sudo,omitempty"` // non-nil will override
	Spec          *Spec     `yaml:"spec"`
	Target        string    `yaml:"target"`
}

func (d *Deployment) Validate() {
	if d.ID.String() == zeroUUIDString {
		d.ID = uuid.Must(uuid.NewRandom())
	}
}

func (d *Deployment) GetID() uuid.UUID {
	return d.ID
}

func (d *Deployment) Hash() (string, error) {
	return Hash(d)
}

func (d *Deployment) HashFile(specs ...*Spec) string {
	spec := d.Spec
	if len(specs) > 0 {
		spec = specs[0]
	}

	return HashFile("deployment", spec)
}

func (d *Deployment) WriteHashLocally() (string, func(), error) {
	return WriteHashLocally(d)
}

func (d *Deployment) LocalAndRemoteHashesMatch(c connection_types.Deployable) (bool, error) {
	return LocalAndRemoteHashesMatch("deployment", d, c)
}

func (d *Deployment) WriteAttemptedHashToRemote(c connection_types.Deployable) error {
	return WriteAttemptedHashToRemote(d, c)
}

func (d *Deployment) CommitRemoteHash(c connection_types.Deployable) error {
	return CommitRemoteHash(d, c)
}
