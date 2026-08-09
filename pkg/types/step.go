package types

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
)

type Step struct {
	Script         string `yaml:"script"`
	ScriptWithSudo string `yaml:"script_with_sudo"`
}

func HashSteps(steps []Step) (string, error) {
	data, err := json.Marshal(steps)
	if err != nil {
		return "", fmt.Errorf("failed to hash steps because %s", err)
	}

	hash := sha256.Sum256(data)

	return hex.EncodeToString(hash[:]), nil
}
