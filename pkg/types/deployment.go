package types

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/google/uuid"
	"github.com/initialed85/deployed/pkg/connection/connection_types"
	"github.com/initialed85/deployed/pkg/helpers/misc"
)

var zeroUUID = uuid.UUID{}
var zeroUUIDString = zeroUUID.String()

type Deployment struct {
	ID            uuid.UUID `yaml:"id"`
	ForceWithSudo *bool     `yaml:"force_sudo,omitempty"` // non-nil will override
	Spec          *Spec     `yaml:"spec"`
	Target        string    `yaml:"target"`
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
		ID:            zeroUUID,
		ForceWithSudo: d.ForceWithSudo,
		Spec:          d.Spec,
		Target:        d.Target,
	}

	return Hash(c)
}

func (d *Deployment) HashFile() string {
	return fmt.Sprintf("deployed-%s-hash.txt", d.Spec.GetName())
}

func (d *Deployment) WriteHashLocally() (string, func(), error) {
	hash, err := d.Hash()
	if err != nil {
		return "", misc.Noop, err
	}

	hashPath := fmt.Sprintf("/tmp/%s.local-%s", d.HashFile(), d.ID)

	err = os.WriteFile(hashPath, []byte(hash), 0o777)
	if err != nil {
		return "", misc.Noop, err
	}

	cleanup := func() {
		_ = os.Remove(hashPath)
	}

	return hashPath, cleanup, nil
}

func (d *Deployment) LocalAndRemoteHashesMatch(c connection_types.Deployable) (bool, error) {
	remoteHashPath := d.HashFile()

	out, _, _ := c.RunCommand(fmt.Sprintf("test -f '%s' && cat '%s'", remoteHashPath, remoteHashPath))

	remoteHash := strings.TrimSpace(out)

	localHash, err := d.Hash()
	if err != nil {
		return false, err
	}

	hashesMatch := localHash == remoteHash

	if hashesMatch {
		log.Printf("local and remote deployment hashes match (%s vs %s)", localHash, remoteHash)
		return hashesMatch, nil
	}

	log.Printf("local and remote deployment hashes don't match (%s vs %s)", localHash, remoteHash)
	return hashesMatch, nil
}

func (d *Deployment) WriteAttemptedHashToRemote(c connection_types.Deployable) error {
	localHash, err := d.Hash()
	if err != nil {
		return err
	}

	remoteAttemptedHashPath := fmt.Sprintf("%s.attempted-%s", d.HashFile(), d.ID)

	_, _, err = c.RunCommand(fmt.Sprintf("echo '%s' > '%s'", localHash, remoteAttemptedHashPath))
	if err != nil {
		return err
	}

	return nil
}

func (d *Deployment) CommitRemoteHash(c connection_types.Deployable) error {
	remoteAttemptedHashPath := fmt.Sprintf("%s.attempted-%s", d.HashFile(), d.ID)

	remoteHashPath := d.HashFile()

	_, _, err := c.RunCommand(fmt.Sprintf("mv -fv %s %s", remoteAttemptedHashPath, remoteHashPath))
	if err != nil {
		return err
	}

	return nil
}
