#!/bin/bash

set -e

# k3s is served from the private net by the lab mirror (./env.sh up starts it, see tools/mirror); GitHub is
# the fallback for a lab that was brought up by hand.
INSTALLER=https://get.k3s.io
ARTIFACT_URL=https://github.com/k3s-io/k3s/releases/download
if curl -fsS --max-time 5 -o /dev/null http://192.168.56.1:8080/get.k3s.io; then
	INSTALLER=http://192.168.56.1:8080/get.k3s.io
	ARTIFACT_URL=http://192.168.56.1:8080/k3s-io/k3s/releases/download
fi

curl -sfL "$INSTALLER" | K3S_TOKEN=some-token K3S_KUBECONFIG_MODE=644 INSTALL_K3S_VERSION=v1.36.2+k3s1 INSTALL_K3S_ARTIFACT_URL="$ARTIFACT_URL" sh -s - agent --server https://192.168.56.11:6443
