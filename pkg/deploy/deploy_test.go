package deploy

import (
	"fmt"
	"sync"
	"testing"

	"github.com/initialed85/deployed/pkg/helpers/pointers"
	"github.com/initialed85/deployed/pkg/types"
	"github.com/stretchr/testify/require"
)

func TestDeploy(t *testing.T) {
	t.Run("K3sMaster", func(t *testing.T) {
		target := "user2:Password2@192.168.56.11:22"

		script := `
#!/bin/bash

set -e

IP="$(hostname -i | cut -d ' ' -f 2)"

curl -sfL https://get.k3s.io | K3S_TOKEN=some-token K3S_KUBECONFIG_MODE=644 INSTALL_K3S_VERSION=v1.36.2+k3s1 sh -s - server --advertise-address="${IP}" --node-ip="${IP}"

mkdir -p /home/user2/.kube

cat /etc/rancher/k3s/k3s.yaml | sed 's/127.0.0.1/192.168.56.11/g' > /home/user2/.kube/config

chown -fR ${USER}:${USER} /home/user2/.kube
`

		steps := []types.Step{
			{
				WithSudo: pointers.Ptr(true),
				Script:   script,
				Downloads: []types.Download{{
					Remote: "/home/user2/.kube",
					Local:  "/tmp/home-user2-kube",
				}},
			},
		}

		spec := (&types.Spec{
			Steps: steps,
		}).TestOnlySetName("k3s-server")

		deployment := types.Deployment{
			Spec:   spec,
			Target: target,
		}

		tookAction, err := Deploy(deployment)
		require.NoError(t, err)
		require.True(t, tookAction, `warning: this test will only pass the first time- you probably want to run "./env.sh restore" first or run your tests with "./test.sh restore""`)

		tookAction, err = Deploy(deployment)
		require.NoError(t, err)
		require.False(t, tookAction)
	})

	wg := new(sync.WaitGroup)

	for i := range 3 {
		wg.Go(func() {
			t.Run(fmt.Sprintf("K3sAgents%d", i+1), func(t *testing.T) {
				target := fmt.Sprintf("user2:Password2@192.168.56.1%d:22", i+2)

				script := `
#!/bin/bash

set -e

curl -sfL https://get.k3s.io | K3S_TOKEN=some-token K3S_KUBECONFIG_MODE=644 INSTALL_K3S_VERSION=v1.36.2+k3s1 sh -s - agent --server https://192.168.56.11:6443
`

				steps := []types.Step{
					{
						WithSudo: pointers.Ptr(true),
						Uploads: []types.Upload{{
							Local:  "/tmp/home-user2-kube",
							Remote: "/home/user2/.kube",
						}},
						Script: script,
					},
				}

				spec := (&types.Spec{
					Steps: steps,
				}).TestOnlySetName(fmt.Sprintf("k3s-agent-%d", i+1))

				deployment := types.Deployment{
					Spec:   spec,
					Target: target,
				}

				tookAction, err := Deploy(deployment)
				require.NoError(t, err)
				require.True(t, tookAction)

				tookAction, err = Deploy(deployment)
				require.NoError(t, err)
				require.False(t, tookAction)
			})
		})
	}

	wg.Wait()
}
