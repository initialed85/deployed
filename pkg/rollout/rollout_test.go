package rollout

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRollout(t *testing.T) {
	err := Rollout("../../test/k3s.yaml")
	require.NoError(t, err)
}
