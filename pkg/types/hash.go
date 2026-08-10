package types

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
)

func Hash(obj any) (string, error) {
	data, err := json.Marshal(obj)
	if err != nil {
		return "", fmt.Errorf("failed to hash %#+v because %s", obj, err)
	}

	hash := sha256.Sum256(data)

	return hex.EncodeToString(hash[:]), nil
}
