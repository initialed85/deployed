#!/bin/bash

set -e

IP="$(hostname -i | cut -d ' ' -f 2)"

curl -sfL https://get.k3s.io | K3S_TOKEN=some-token K3S_KUBECONFIG_MODE=644 INSTALL_K3S_VERSION=v1.36.2+k3s1 sh -s - server --advertise-address="${IP}" --node-ip="${IP}"

mkdir -p /home/user2/.kube

cat /etc/rancher/k3s/k3s.yaml | sed "s/127.0.0.1/${IP}/g" >/home/user2/.kube/config

# shellcheck disable=SC2086
chown -fR ${USER}:${USER} /home/user2/.kube
