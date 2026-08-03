#!/bin/bash
# env.sh - platform-aware wrapper around docker compose + the Vagrantfile.
#
# Usage:
#   ./env.sh up                  docker compose up --build, then all nodes
#   ./env.sh up node2            docker compose up --build, then a single node
#   ./env.sh down                halt all nodes, then docker compose down
#   ./env.sh down node2          halt a single node (leaves docker compose up)
#   ./env.sh destroy             destroy all nodes (and their disks)
#   ./env.sh status              show vagrant status
#   ./env.sh provision           re-run the provisioner on all nodes
#   ./env.sh ssh <node> [user]   ssh into a node as <user> (default: user1)
#   ./env.sh port <node>         print the host-side SSH port for a node
#   ./env.sh provider            print the auto-detected provider
#
# Override the auto-detected provider:
#   ENV_PROVIDER=libvirt ./env.sh up
#   ENV_PROVIDER=virtualbox ./env.sh up

set -euo pipefail

# --- platform detection ----------------------------------------------------

os="$(uname -s)"
arch="$(uname -m)"
case "$os" in
Darwin) platform="macos" ;;
Linux) platform="linux" ;;
MINGW* | MSYS* | CYGWIN*) platform="windows" ;;
*) platform="${os,,}" ;;
esac
case "$arch" in
arm64 | aarch64) arch="arm64" ;;
x86_64 | amd64) arch="amd64" ;;
esac

# Providers that ship with Vagrant (no plugin needed). Everything else
# requires a vagrant-<provider> plugin to be installed to be usable.
BUILTIN_PROVIDERS="virtualbox hyperv docker"

# plugin_for() -> echoes the plugin name for a non-built-in provider,
# or empty if it isn't a known plugin-required provider.
plugin_for() {
	case "$1" in
	libvirt) echo "vagrant-libvirt" ;;
	qemu) echo "vagrant-qemu" ;;
	parallels) echo "vagrant-parallels" ;;
	vmware_desktop) echo "vagrant-vmware-desktop" ;;
	*) echo "" ;;
	esac
}

# provider_ready() -> 0 if the provider is actually usable right now, not just
# installed. Plugin install alone isn't enough: e.g. libvirt needs a running
# daemon, parallels needs a licensed Parallels, etc.
provider_ready() {
	local p="$1"
	case "$p" in
	libvirt)
		# libvirt daemon reachable on the default system URI?
		virsh -c qemu:///system uri >/dev/null 2>&1 && return 0
		# macOS (macports/homebrew) often uses a per-user socket; try that too.
		[[ -S /var/run/libvirt/libvirt-sock ]] ||
			[[ -S /opt/local/var/run/libvirt/libvirt-sock ]] ||
			[[ -S "$HOME/.libvirt/libvirt-sock" ]] && return 0
		return 1
		;;
	parallels)
		# Parallels CLI present *and* the VM service responding.
		command -v prlctl >/dev/null 2>&1 && prlctl list -a >/dev/null 2>&1
		;;
	qemu)
		# QEMU binary is qemu-system-aarch64 on arm64 hosts (uname says arm64).
		local qarch="amd64"
		[[ "$arch" == "arm64" ]] && qarch="aarch64"
		command -v "qemu-system-${qarch}" >/dev/null 2>&1
		;;
	*)
		return 0
		;;
	esac
}

detect_provider() {
	if [[ -n "${ENV_PROVIDER:-}" ]]; then
		echo "${ENV_PROVIDER}"
		return
	fi
	# Candidate preference order per host.
	# macOS / Apple Silicon: libvirt (free) > qemu > virtualbox > parallels
	# macOS / Intel:         libvirt > virtualbox > qemu
	# Linux:                 libvirt > virtualbox
	# Windows:               virtualbox
	local candidates
	case "$platform/$arch" in
	macos/arm64) candidates="libvirt qemu virtualbox parallels" ;;
	macos/amd64) candidates="libvirt virtualbox qemu" ;;
	linux/*) candidates="libvirt virtualbox" ;;
	windows/*) candidates="virtualbox" ;;
	*) candidates="virtualbox libvirt parallels qemu" ;;
	esac

	local plugins
	plugins="$(vagrant plugin list 2>/dev/null || true)"

	local p plugin
	for p in $candidates; do
		if [[ " $BUILTIN_PROVIDERS " == *" $p "* ]]; then
			case "$p" in
			virtualbox) [[ -x "$(command -v VBoxManage 2>/dev/null)" ]] && provider_ready "$p" && {
				echo "$p"
				return
			} ;;
			hyperv) [[ "$platform" == "windows" ]] && {
				echo "$p"
				return
			} ;;
			docker) command -v docker >/dev/null 2>&1 && {
				echo "$p"
				return
			} ;;
			esac
			continue
		fi
		plugin="$(plugin_for "$p")"
		[[ -z "$plugin" ]] && continue
		if printf '%s\n' "$plugins" | grep -q "^${plugin} " && provider_ready "$p"; then
			echo "$p"
			return
		fi
	done
	# Fallback: let Vagrant choose its default.
	echo ""
}

provider="$(detect_provider)"

# --- helpers ---------------------------------------------------------------

NODE_PORTS=(node1:12221 node2:12222 node3:12223 node4:12224)

port_for_node() {
	local node="$1"
	for entry in "${NODE_PORTS[@]}"; do
		if [[ "${entry%%:*}" == "$node" ]]; then
			echo "${entry##*:}"
			return
		fi
	done
	echo "env.sh: unknown node '$node' (expected: ${NODE_PORTS[*]%%:*})" >&2
	exit 1
}

vagrant_up() {
	if [[ -n "$provider" ]]; then
		vagrant up --provider="$provider" "$@"
	else
		vagrant up "$@"
	fi
}

require_vagrant() {
	command -v vagrant >/dev/null 2>&1 || {
		echo "env.sh: 'vagrant' not found on PATH. Install Vagrant first:" >&2
		echo "  brew install vagrant   # or: https://developer.hashicorp.com/vagrant/install" >&2
		exit 1
	}
}

require_docker() {
	command -v docker >/dev/null 2>&1 || {
		echo "env.sh: 'docker' not found on PATH. Install Docker first:" >&2
		echo "  https://docs.docker.com/get-docker/" >&2
		exit 1
	}
}

# docker compose is only invoked when a compose file is present in the
# project root.  This keeps env.sh usable in repos that are vagrant-only.
compose_file() {
	for f in docker-compose.yaml docker-compose.yml compose.yaml compose.yml; do
		[[ -f "$f" ]] && { echo "$f"; return 0; }
	done
	return 1
}

compose_up() {
	local cf
	cf="$(compose_file)" || return 0
	require_docker
	echo "env.sh: docker compose up --build (detached)"
	docker compose -f "$cf" up -d --build
}

compose_down() {
	local cf
	cf="$(compose_file)" || return 0
	require_docker
	echo "env.sh: docker compose down --volumes --remove-orphans"
	docker compose -f "$cf" down --volumes --remove-orphans
}

# --- commands --------------------------------------------------------------

cmd_up() {
	require_vagrant
	compose_up
	echo "env.sh: bringing up nodes on $platform/$arch via provider=${provider:-<default>}"
	vagrant_up "$@"
}

cmd_down() {
	require_vagrant
	vagrant halt "$@"
	# Only tear down docker compose when halting everything.  A single-node
	# down (e.g. `down node2`) leaves the compose services running so the
	# rest of the test harness stays available.
	if [[ $# -eq 0 ]]; then
		compose_down
	fi
}

cmd_destroy() {
	require_vagrant
	vagrant destroy -f "$@"
}

cmd_status() {
	require_vagrant
	vagrant status "$@"
}

cmd_provision() {
	require_vagrant
	vagrant provision "$@"
}

cmd_ssh() {
	local node="${1:-}"
	local user="${2:-user1}"
	[[ -n "$node" ]] || {
		echo "usage: env.sh ssh <node> [user]" >&2
		exit 1
	}
	local port
	port="$(port_for_node "$node")"
	echo "env.sh: ssh ${user}@localhost:${port} ($node)"
	ssh -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null \
		-p "$port" "${user}@localhost"
}

cmd_port() {
	local node="${1:-}"
	[[ -n "$node" ]] || {
		echo "usage: env.sh port <node>" >&2
		exit 1
	}
	port_for_node "$node"
}

cmd_provider() {
	echo "${provider:-<default>}"
}

cmd_info() {
	echo "platform:  $platform"
	echo "arch:      $arch"
	echo "provider:  ${provider:-<default>}"
	echo "nodes:"
	for entry in "${NODE_PORTS[@]}"; do
		printf "  %s -> localhost:%s\n" "${entry%%:*}" "${entry##*:}"
	done
}

# --- dispatch ---------------------------------------------------------------

usage() {
	awk 'NR==1 && /^#!/ { next } /^#/ { sub(/^# ?/,"  "); print; next } { exit }' "$0"
}

sub="${1:-}"
[[ $# -gt 0 ]] && shift
case "$sub" in
up) cmd_up "$@" ;;
down) cmd_down "$@" ;;
destroy) cmd_destroy "$@" ;;
status) cmd_status "$@" ;;
provision) cmd_provision "$@" ;;
ssh) cmd_ssh "$@" ;;
port) cmd_port "$@" ;;
provider) cmd_provider "$@" ;;
info) cmd_info "$@" ;;
"" | help | -h | --help) usage ;;
*)
	echo "env.sh: unknown command '$sub'" >&2
	usage >&2
	exit 1
	;;
esac
