#!/bin/bash

set -e

curl -sfL https://get.k3s.io | K3S_TOKEN=some-token K3S_KUBECONFIG_MODE=644 INSTALL_K3S_VERSION=v1.36.2+k3s1 sh -s - agent --server https://192.168.56.11:6443
