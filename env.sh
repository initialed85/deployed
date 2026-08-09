#!/bin/bash

set -euo pipefail

#
# platform handling
#

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

BUILTIN_PROVIDERS="virtualbox hyperv docker"

plugin_for() {
	case "$1" in
	libvirt) echo "vagrant-libvirt" ;;
	qemu) echo "vagrant-qemu" ;;
	parallels) echo "vagrant-parallels" ;;
	vmware_desktop) echo "vagrant-vmware-desktop" ;;
	*) echo "" ;;
	esac
}

provider_ready() {
	local p="$1"

	case "$p" in

	libvirt)
		virsh -c qemu:///system uri >/dev/null 2>&1 && return 0
		[[ -S /var/run/libvirt/libvirt-sock ]] ||
			[[ -S /opt/local/var/run/libvirt/libvirt-sock ]] ||
			[[ -S "$HOME/.libvirt/libvirt-sock" ]] && return 0
		return 1
		;;

	parallels)
		command -v prlctl >/dev/null 2>&1 && prlctl list -a >/dev/null 2>&1
		;;

	qemu)
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

	echo ""
}

provider="$(detect_provider)"

#
# helpers
#

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

compose_file() {
	echo "docker-compose.yaml"
	return 0
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
	local args=(-f "$cf")
	[[ -f "docker-compose.restore.yaml" ]] && args+=(-f "docker-compose.restore.yaml")
	echo "env.sh: docker compose down --volumes --remove-orphans"
	docker compose "${args[@]}" down --volumes --remove-orphans
}

vagrant_up() {
	if [[ -n "$provider" ]]; then
		vagrant up --provider="$provider"
	else
		vagrant up
	fi
}

#
# commands
#

cmd_up() {
	require_vagrant

	rm -f "docker-compose.restore.yaml" || true
	rm -f "docker-compose.restore.yaml.bak" || true
	rm -f "$VAGRANT_SNAPSHOT_DIR"/*.save 2>/dev/null || true

	compose_up

	echo "env.sh: bringing up nodes on $platform/$arch via provider=${provider:-<default>}"
	vagrant_up

	echo "env.sh: taking initial snapshot"
	cmd_snapshot
}

cmd_down() {
	require_vagrant
	vagrant destroy -f
	compose_down
	rm -f "$VAGRANT_SNAPSHOT_DIR"/*.save 2>/dev/null || true
	rm -f docker-compose.restore.yaml docker-compose.restore.yaml.bak
}

cmd_vagrant() {
	require_vagrant
	vagrant "$@"
}

# --- snapshot/restore ------------------------------------------------------

VAGRANT_SNAPSHOT_NAME="env-sh"
LIBVIRT_URI="qemu:///system"
VAGRANT_SNAPSHOT_DIR=".vagrant/snapshots"

uuid() {
	if command -v uuidgen >/dev/null 2>&1; then
		uuidgen
	elif [[ -r /proc/sys/kernel/random/uuid ]]; then
		cat /proc/sys/kernel/random/uuid
	else
		printf '%s-%s-%s' "$(date +%s)" "$RANDOM" "$RANDOM"
	fi
}

vagrant_vm_names() {
	vagrant status --machine-readable 2>/dev/null |
		awk -F, '$3 == "provider-name" { print $2 }' |
		sort -u
}

libvirt_domain_for() {
	local vm="$1"
	virsh -c "$LIBVIRT_URI" list --all --name 2>/dev/null |
		grep -E "_${vm}$" | head -1
}

compose_snapshot() {
	local cf bak uuid svc cid tag committed

	cf="$(compose_file)" || {
		echo "env.sh: no docker-compose file; skipping containers"
		return 0
	}

	require_docker

	bak="docker-compose.restore.yaml.bak"
	# shellcheck disable=SC2019
	# shellcheck disable=SC2018
	uuid="$(uuid | tr 'A-Z' 'a-z')"

	rm -f "docker-compose.restore.yaml" "$bak"
	printf 'services:\n' >"$bak"

	committed=0

	while IFS= read -r svc; do
		cid="$(docker compose -f "$cf" ps -q "$svc" 2>/dev/null || true)"
		if [[ -z "$cid" ]]; then
			echo "env.sh: skip '$svc' (not running)"
			continue
		fi
		tag="env-restore:snap-${uuid}-${svc}"
		echo "env.sh: docker commit $svc -> $tag"
		docker commit "$cid" "$tag" >/dev/null
		printf '  %s:\n    image: %s\n' "$svc" "$tag" >>"$bak"
		committed=$((committed + 1))
	done < <(docker compose -f "$cf" config --services)

	if ((committed == 0)); then
		echo "env.sh: no running containers snapshotted; removed restore overlay"
		rm -f "$bak"
	else
		echo "env.sh: docker snapshot saved -> $bak ($committed service(s))"
	fi
}

compose_restore() {
	local cf bak cur

	cf="$(compose_file)" || {
		echo "env.sh: no docker-compose file; skipping containers"
		return 0
	}

	require_docker

	bak="docker-compose.restore.yaml.bak"
	cur="docker-compose.restore.yaml"
	if [[ ! -f "$bak" ]]; then
		echo "env.sh: no docker snapshot found (missing $bak)" >&2
		return 1
	fi

	cp -f "$bak" "$cur"

	echo "env.sh: docker compose up (no build, from snapshot $cur)"
	docker compose -f "$cf" -f "$cur" stop -t0 2>/dev/null || true
	docker compose -f "$cf" -f "$cur" up -d
}

vagrant_snapshot() {
	require_vagrant
	if [[ "$provider" == "libvirt" ]]; then
		libvirt_snapshot
	else
		echo "env.sh: vagrant snapshot save $VAGRANT_SNAPSHOT_NAME"
		vagrant snapshot delete "$VAGRANT_SNAPSHOT_NAME" >/dev/null 2>&1 || true
		vagrant snapshot save "$VAGRANT_SNAPSHOT_NAME"
	fi
}

vagrant_restore() {
	require_vagrant
	if [[ "$provider" == "libvirt" ]]; then
		libvirt_restore
	else
		echo "env.sh: vagrant snapshot restore $VAGRANT_SNAPSHOT_NAME"
		vagrant snapshot restore "$VAGRANT_SNAPSHOT_NAME"
	fi
}

libvirt_snapshot() {
	local snapdir="$VAGRANT_SNAPSHOT_DIR" vm domain state disk saved

	mkdir -p "$snapdir"

	saved=0

	while IFS= read -r vm; do
		[[ -z "$vm" ]] && continue
		domain="$(libvirt_domain_for "$vm")"
		if [[ -z "$domain" ]]; then
			echo "env.sh: skip $vm (no libvirt domain)"
			continue
		fi
		state="$(virsh -c "$LIBVIRT_URI" domstate "$domain" 2>/dev/null)"
		if [[ "$state" != "running" ]]; then
			echo "env.sh: skip $vm ($state)"
			continue
		fi
		echo "env.sh: virsh save $domain -> $snapdir/${vm}.save"
		virsh -c "$LIBVIRT_URI" save "$domain" "$snapdir/${vm}.save"
		disk="$(virsh -c "$LIBVIRT_URI" domblklist "$domain" --details 2>/dev/null |
			awk '$2 == "disk" { print $NF }')"
		if [[ -n "$disk" ]]; then
			echo "env.sh: qemu-img snapshot -c $VAGRANT_SNAPSHOT_NAME $disk"
			sudo qemu-img snapshot -c "$VAGRANT_SNAPSHOT_NAME" "$disk"
		fi
		echo "env.sh: virsh restore $snapdir/${vm}.save"
		virsh -c "$LIBVIRT_URI" restore "$snapdir/${vm}.save"
		saved=$((saved + 1))
	done < <(vagrant_vm_names)

	if ((saved == 0)); then
		echo "env.sh: no running VMs snapshotted"
		return 1
	fi

	echo "env.sh: vagrant snapshot saved ($saved VM(s)) -> $snapdir/"
}

libvirt_restore() {
	local snapdir="$VAGRANT_SNAPSHOT_DIR" savefile vm domain state disk restored

	if [[ ! -d "$snapdir" ]]; then
		echo "env.sh: no vagrant snapshot found (missing $snapdir)" >&2
		return 1
	fi

	for savefile in "$snapdir"/*.save; do
		[[ -f "$savefile" ]] || continue
		vm="$(basename "$savefile" .save)"
		domain="$(libvirt_domain_for "$vm")"
		if [[ -n "$domain" ]]; then
			state="$(virsh -c "$LIBVIRT_URI" domstate "$domain" 2>/dev/null)"
			if [[ "$state" == "running" ]]; then
				echo "env.sh: virsh destroy $domain (running)"
				virsh -c "$LIBVIRT_URI" destroy "$domain" >/dev/null
			fi
		fi
	done

	for savefile in "$snapdir"/*.save; do
		[[ -f "$savefile" ]] || continue
		vm="$(basename "$savefile" .save)"
		domain="$(libvirt_domain_for "$vm")"
		[[ -n "$domain" ]] || continue
		disk="$(virsh -c "$LIBVIRT_URI" domblklist "$domain" --details 2>/dev/null |
			awk '$2 == "disk" { print $NF }')"
		if [[ -n "$disk" ]]; then
			echo "env.sh: qemu-img snapshot -a $VAGRANT_SNAPSHOT_NAME $disk"
			sudo qemu-img snapshot -a "$VAGRANT_SNAPSHOT_NAME" "$disk"
		fi
	done

	restored=0

	for savefile in "$snapdir"/*.save; do
		[[ -f "$savefile" ]] || continue
		echo "env.sh: virsh restore $savefile"
		virsh -c "$LIBVIRT_URI" restore "$savefile"
		restored=$((restored + 1))
	done

	if ((restored == 0)); then
		echo "env.sh: no save files in $snapdir" >&2
		return 1
	fi

	echo "env.sh: vagrant restored ($restored VM(s)) from $snapdir/"
}

cmd_snapshot() {
	compose_snapshot
	vagrant_snapshot
}

cmd_restore() {
	compose_restore
	vagrant_restore
}

#
# entrypoint
#

usage() {
	awk 'NR==1 && /^#!/ { next } /^#/ { sub(/^# ?/,"  "); print; next } { exit }' "$0"
}

sub="${1:-}"
case "$sub" in
up) cmd_up ;;
down) cmd_down ;;
snapshot) cmd_snapshot ;;
restore) cmd_restore ;;
vagrant) shift; cmd_vagrant "$@" ;;
"" | help | -h | --help) usage ;;
*)
	echo "env.sh: unknown command '$sub'" >&2
	usage >&2
	exit 1
	;;
esac

echo -e "\nenv.sh: done."
