package types

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/google/uuid"
	"github.com/initialed85/deployed/pkg/connection/connection_types"
	"github.com/initialed85/deployed/pkg/helpers/misc"
)

func Hash(obj any) (string, error) {
	data, err := json.Marshal(obj)
	if err != nil {
		return "", fmt.Errorf("failed to hash %#+v because %s", obj, err)
	}

	rawHash := sha256.Sum256(data)

	hash := hex.EncodeToString(rawHash[:])

	return hash, nil
}

func HashFile(file *os.File) (string, error) {
	hasher := sha256.New()

	_, err := hasher.Write([]byte(file.Name()))
	if err != nil {
		return "", err
	}

	_, err = io.Copy(hasher, file)
	if err != nil {
		return "", err
	}

	rawHash := hasher.Sum(nil)

	hash := hex.EncodeToString(rawHash[:])

	return hash, nil
}

type Hashable interface {
	GetID() uuid.UUID
	Hash() (string, error)
	GetHashFileName(specs ...*Spec) string
}

func GetHashFileName(hashType string, spec *Spec) string {
	if spec == nil {
		panic("spec unexpectedly nil for HashFile")
	}

	return fmt.Sprintf("deployed-%s-%s.hash", hashType, spec.GetName())
}

func WriteHashLocally(h Hashable, specs ...*Spec) (string, func(), error) {
	hash, err := h.Hash()
	if err != nil {
		return "", misc.Noop, err
	}

	hashPath := fmt.Sprintf("/tmp/%s.local-%s", h.GetHashFileName(specs...), h.GetID())

	err = os.WriteFile(hashPath, []byte(hash), 0o777)
	if err != nil {
		return "", misc.Noop, err
	}

	cleanup := func() {
		_ = os.Remove(hashPath)
	}

	return hashPath, cleanup, nil
}

func LocalAndRemoteHashesMatch(hashType string, h Hashable, c connection_types.Deployable, specs ...*Spec) (bool, error) {
	remoteHashPath := h.GetHashFileName(specs...)

	out, _, _ := c.RunCommand(fmt.Sprintf("test -f '%s' && cat '%s'", remoteHashPath, remoteHashPath))

	remoteHash := strings.TrimSpace(out)

	localHash, err := h.Hash()
	if err != nil {
		return false, err
	}

	hashesMatch := localHash == remoteHash

	if hashesMatch {
		log.Printf("local and remote %s hashes match (%s vs %s)", hashType, localHash, remoteHash)
		return hashesMatch, nil
	}

	log.Printf("local and remote %s hashes don't match (%s vs %s)", hashType, localHash, remoteHash)
	return hashesMatch, nil
}

func WriteAttemptedHashToRemote(h Hashable, c connection_types.Deployable, specs ...*Spec) error {
	localHash, err := h.Hash()
	if err != nil {
		return err
	}

	remoteAttemptedHashPath := fmt.Sprintf("%s.attempted-%s", h.GetHashFileName(specs...), h.GetID())

	_, _, err = c.RunCommand(fmt.Sprintf("echo '%s' > '%s'", localHash, remoteAttemptedHashPath))
	if err != nil {
		return err
	}

	return nil
}

func CommitRemoteHash(h Hashable, c connection_types.Deployable, specs ...*Spec) error {
	remoteAttemptedHashPath := fmt.Sprintf("%s.attempted-%s", h.GetHashFileName(specs...), h.GetID())

	remoteHashPath := h.GetHashFileName(specs...)

	_, _, err := c.RunCommand(fmt.Sprintf("mv -fv %s %s", remoteAttemptedHashPath, remoteHashPath))
	if err != nil {
		return err
	}

	return nil
}
