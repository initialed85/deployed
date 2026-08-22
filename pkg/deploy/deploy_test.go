package deploy

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/initialed85/deployed/pkg/connection"
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
		// TODO(initialed85): may as well not have it lol- deals with the fact you might be doing a warm test run (i.e. already deployed)
		require.True(t, tookAction == true || tookAction == false)

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
				// TODO(initialed85): may as well not have it lol- deals with the fact you might be doing a warm test run (i.e. already deployed)
				require.True(t, tookAction == true || tookAction == false)

				tookAction, err = Deploy(deployment)
				require.NoError(t, err)
				require.False(t, tookAction)
			})
		})
	}

	wg.Wait()
}

func TestDeployModes(t *testing.T) {
	modes := []struct {
		name       string
		upload     bool
		script     bool
		download   bool
		tookAction bool
	}{
		{name: "Full", upload: true, script: true, download: true, tookAction: true},
		{name: "ScriptsOnly", script: true, tookAction: true},
		{name: "UploadsOnly", upload: true, tookAction: true},
		{name: "DownloadsOnly", download: true},
	}

	for _, mode := range modes {
		mode := mode
		t.Run(mode.name, func(t *testing.T) {
			target := "user2:Password2@localhost:2221"
			specName := fmt.Sprintf("deploy-modes-%s", uuid.NewString())
			remoteDir := fmt.Sprintf("/home/user2/%s", specName)
			remoteUpload := fmt.Sprintf("%s/upload.txt", remoteDir)
			remoteScript := fmt.Sprintf("%s/script.txt", remoteDir)
			remoteDownload := fmt.Sprintf("%s/download.txt", remoteDir)
			localDir := t.TempDir()
			localUpload := filepath.Join(localDir, "upload.txt")
			localDownload := filepath.Join(localDir, "download.txt")

			err := os.WriteFile(localUpload, []byte("upload content\n"), 0o644)
			require.NoError(t, err)

			c, err := connection.OpenSSH("localhost", 2221, "user2", "Password2")
			require.NoError(t, err)
			defer c.Close()
			defer func() {
				_, _, _ = c.RunCommand(fmt.Sprintf("rm -rf '%s' deployed-deployment-%s* deployed-step-%s*", remoteDir, specName, specName))
			}()

			_, _, err = c.RunCommand(fmt.Sprintf("mkdir -p '%s'", remoteDir))
			require.NoError(t, err)

			step := types.Step{}

			if mode.upload {
				step.Uploads = []types.Upload{{
					Local:  localUpload,
					Remote: remoteUpload,
				}}
			}

			if mode.script {
				step.Script = fmt.Sprintf(`#!/bin/bash
set -e

if [ -e '%s' ]; then
	exit 1
fi
%s
printf 'script content\n' > '%s'
`, remoteScript, func() string {
					if mode.upload {
						return fmt.Sprintf("test \"$(cat '%s')\" = 'upload content'", remoteUpload)
					}

					return "true"
				}(), remoteScript)
			}

			if mode.download {
				step.Downloads = []types.Download{{
					WithSudo: pointers.Ptr(false),
					Remote:   remoteDownload,
					Local:    localDownload,
				}}

				_, _, err = c.RunCommand(fmt.Sprintf("printf 'download content\\n' > '%s'", remoteDownload))
				require.NoError(t, err)
			}

			spec := &types.Spec{
				Name:     specName,
				WithSudo: pointers.Ptr(true),
				Steps:    []types.Step{step},
			}
			deployment := types.Deployment{
				Spec:   spec,
				Target: target,
			}

			tookAction, err := Deploy(deployment)
			require.NoError(t, err)
			require.Equal(t, mode.tookAction, tookAction)

			if mode.upload {
				stdout, _, err := c.RunCommand(fmt.Sprintf("cat '%s'", remoteUpload))
				require.NoError(t, err)
				require.Equal(t, "upload content\n", stdout)
			}

			if mode.script {
				stdout, _, err := c.RunCommand(fmt.Sprintf("cat '%s'", remoteScript))
				require.NoError(t, err)
				require.Equal(t, "script content\n", stdout)
			}

			if mode.download {
				contents, err := os.ReadFile(localDownload)
				require.NoError(t, err)
				require.Equal(t, "download content\n", string(contents))
			}

			if mode.download {
				err = os.Remove(localDownload)
				require.NoError(t, err)
			}

			tookAction, err = Deploy(deployment)
			require.NoError(t, err)
			require.False(t, tookAction)

			if mode.download {
				contents, err := os.ReadFile(localDownload)
				require.NoError(t, err)
				require.Equal(t, "download content\n", string(contents))
			}
		})
	}
}
