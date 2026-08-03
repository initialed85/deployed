package deploy

import (
	"testing"

	"github.com/initialed85/deployed/pkg/types"
	"github.com/stretchr/testify/require"
)

func TestDeploy(t *testing.T) {
	t.Run("Postgres", func(t *testing.T) {
		target := "user2:Password2@localhost:12221"

		scriptWithSudo := `
# if ps aux | grep postgres | grep -v grep; then
# 	exit 0
# fi	

apt-get update
apt-get install -y postgresql-18
`

		steps := []types.Step{
			{
				ScriptWithSudo: scriptWithSudo,
			},
		}

		err := Deploy(target, steps)
		require.NoError(t, err)
	})

	t.Run("K3sMaster", func(t *testing.T) {
		t.Skip() // TODO

		target := "user2:Password2@localhost:12221"

		scriptWithSudo := `
if ps aux | grep postgres | grep -v grep; then
	exit 0
fi	

apt-get update
apt-get install -y postgresql-18
`

		steps := []types.Step{
			{
				ScriptWithSudo: scriptWithSudo,
			},
		}

		err := Deploy(target, steps)
		require.NoError(t, err)
	})
}
