package deploy

import (
	"testing"

	"github.com/initialed85/deployed/pkg/types"
	"github.com/stretchr/testify/require"
)

func TestDeploy(t *testing.T) {
	t.Run("Postgres", func(t *testing.T) {
		target := "user2:Password2@192.168.56.11:22"

		scriptWithSudo := `
#!/bin/bash

set -e

apt-get update
apt-get install -y ca-certificates curl
install -m 0755 -d /etc/apt/keyrings
curl -fsSL https://download.docker.com/linux/ubuntu/gpg -o /etc/apt/keyrings/docker.asc
chmod a+r /etc/apt/keyrings/docker.asc

echo \
	"deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.asc] https://download.docker.com/linux/ubuntu \
  $(. /etc/os-release && echo "${UBUNTU_CODENAME:-$VERSION_CODENAME}") stable" |
	tee /etc/apt/sources.list.d/docker.list >/dev/null

apt-get update
apt-get install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin

docker run -d --restart=always --name postgres -e 'POSTGRES_PASSWORD=NoCloud!11' -p 5432:5432 -v /var/lib/postgresql:/var/lib/postgresql postgres:18
`

		steps := []types.Step{
			{
				ScriptWithSudo: scriptWithSudo,
			},
		}

		tookAction, err := Deploy(target, steps)
		require.NoError(t, err)
		require.True(t, tookAction)

		tookAction, err = Deploy(target, steps)
		require.NoError(t, err)
		require.False(t, tookAction)
	})

	t.Run("K3sMaster", func(t *testing.T) {
		t.Skipf("TODO: hangs trying to install; not sure if resources, weird virtualization thing, Postgres datastore too slow- no idea")

		target := "user2:Password2@192.168.56.12:22"

		scriptWithSudo := `
#!/bin/bash

set -e

curl -sfL https://get.k3s.io | K3S_TOKEN=some-token K3S_KUBECONFIG_MODE=644 INSTALL_K3S_VERSION=v1.36.2+k3s1 sh -s - server \
    --datastore-endpoint='postgres://postgres:NoCloud!11@192.168.56.11:5432/k3s_dev' \
    --flannel-backend=none \
    --disable-network-policy=true \
    --disable=kube-proxy \
    --cluster-cidr=10.42.0.0/16 \
    --service-cidr=10.43.0.0/16 \
    --advertise-address="$(hostname -i | cut -d ' ' -f 2)" \
    --node-ip="$(hostname -i | cut -d ' ' -f 2)" \
    --kubelet-arg="node-status-update-frequency=1s" \
    --kube-controller-manager-arg="node-monitor-period=2s" \
    --kube-controller-manager-arg="node-monitor-grace-period=8s" \
    --kube-apiserver-arg="default-not-ready-toleration-seconds=8" \
    --kube-apiserver-arg="default-unreachable-toleration-seconds=8"

mkdir -p ~/.kube
cp /etc/rancher/k3s/k3s.yaml ~/.kube/config && chown -fR ${USER}:${USER} ~/.kube/config
	
CILIUM_CLI_VERSION=$(curl -s https://raw.githubusercontent.com/cilium/cilium-cli/main/stable.txt)
CLI_ARCH=amd64
if [ "$(uname -m)" = "aarch64" ]; then CLI_ARCH=arm64; fi
curl -L --fail --remote-name-all https://github.com/cilium/cilium-cli/releases/download/${CILIUM_CLI_VERSION}/cilium-linux-${CLI_ARCH}.tar.gz{,.sha256sum}
tar xzvfC cilium-linux-${CLI_ARCH}.tar.gz /usr/local/bin
rm -fv cilium-linux-${CLI_ARCH}.tar.gz
	
cilium install \
    --version 1.19.6 \
    --namespace kube-system \
    --set routingMode=tunnel\ \
    --set tunnelProtocol=geneve \
    --set kubeProxyReplacement=true \
    --set loadBalancer.mode=dsr \
    --set loadBalancer.dsrDispatch=geneve \
    --set "k8sServiceHost=$(hostname -i | cut -d ' ' -f 2)" \
    --set k8sServicePort=6443 \
    --set=ipam.operator.clusterPoolIPv4PodCIDRList="10.42.0.0/16" \
    --set ipv6.enabled=false
	
cilium status --wait
`

		steps := []types.Step{
			{
				ScriptWithSudo: scriptWithSudo,
			},
		}

		tookAction, err := Deploy(target, steps)
		require.NoError(t, err)
		require.True(t, tookAction)

		tookAction, err = Deploy(target, steps)
		require.NoError(t, err)
		require.False(t, tookAction)
	})
}
