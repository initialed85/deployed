# deployed

## Goal

A simple way to provision software on Linux servers, via SSH.

## Concepts

### Configuration

- Step
  - Some files or folders to transfer
  - A script to run
- Spec
  - A collection of Steps
- Target
  - A machine to provision (via SSH)
- Pool
  - A collection of Targets
- Workspace
  - A collection of Pools
  - The mapping of Pools to a Spec

## Runtime

- Deployment
  - The mapping of a Spec to a Target

## Tasks

### Roadmap

- [WIP] Build local layer
  - [DONE] Be able to run commands
  - [DONE] Be able to run commands with sudo
  - [DONE] Be able to copy files
  - [DONE] Be able to copy files and move them to a privileged location with sudo
  - [DONE] Be able to copy folders
  - [DONE] Be able to copy folders and move them to a privileged location with sudo
  - [TODO] Expose local layer via an app entrypoint (as a low-level escape hatch)
- [WIP] Build SSH layer
  - [DONE] Be able to run commands
  - [DONE] Be able to run commands with sudo
  - [DONE] Be able to transfer files via SCP
  - [DONE] Be able to transfer files via SCP and move them to a privileged location with sudo
  - [DONE] Be able to transfer folders via SCP
  - [DONE] Be able to transfer folders via SCP and move them to a privileged location with sudo
  - [TODO] Have support for SSH compression ([it's complicated and has been since 20191](https://github.com/golang/go/issues/31369))
  - [TODO] Expose SSH layer via an app entrypoint (as a low-level escape hatch)
- [WIP] Build deployment layer
  - [DONE] Be able to run a number of Steps against a Target
  - [DONE] Store a hash of the last run Steps on a Target for idempotency
  - [DONE] Be able to specify one or more Upload files or folders for a Target
  - [DONE] Be able to specify one or more Download files or folders for a Target
  - [TODO] Expose deployment layer via an app entrypoint
- [WIP] Build workspace layer
  - [DONE] Be able to define Specs (collection of Steps)
  - [DONE] Be able to define Pools (collection of Targets)
  - [DONE] Be able to define Mappings (of Pools to Specs)
  - [DONE] Have it all driven by YAML files
  - [DONE] Support naming Downloads
  - [DONE] Expose workspace layer via an app entrypoint
  - [TODO] Split screen log monitor (lazy tmux?)
  - [TODO] Cross-session env var support
    - [TODO] Inject some useful implicit ones; e.g. `DEPLOYED_META_HOST`, `DEPLOYED_META_USER`, `DEPLOYED_META_OS`, `DEPLOYED_META_ARCH`
    - [TODO] Support exporting and reusing private dynamic env vars e.g. `DEPLOYED_PRIVATE_*` (protected from duplicate clashes)
      - Available during the entire rollout, once exported
      - Locally scoped by target
    - [TODO] Support exporting and reusing public dynamic env vars e.g. `DEPLOYED_PUBLIC_*` (protected from duplicate clashes, exported to host env on completion)
      - Available during the entire rollout, once exported
      - Globally scoped

### Debt

- [TODO] Bundle the state in once place locally and remotely
- [TODO] Fix portability issues re: some reliance on shell commands locally / remotely
- [TODO] Have `env.sh` wait until env is ready with a poll (instead of a 1s sleep)
- [TODO] Support for SSH keys as well as passwords (how will this work with sudo?)
- [TODO] Don't just stub out SSH host key validation
- [TODO] Fix the hacked in ${USER} env handling
- [TODO] Have a common abstraction for the Local and SSH connections

### Bugs

- [DONE] Deal with potential local file path collisions for named downloads

## Usage

See [test/k3s.yaml](test/k3s.yaml) for an example configuration (this is used by the tests).

Assuming you've already built with `go build -o ./deployed .`:

```shell
./deployed rollout test/k3s.yaml
```

The master node will deploy first, followed by the agent nodes (concurrently).

The Kube config will be pulled back to `/tmp/home-user2-dot-kube/config` so you can run something like this:

```shell
KUBECONFIG=/tmp/home-user2-dot-kube/config kubectl get -o wide nodes
```

The same Kube config is pushed out to all agents so they can use `kubectl` as well.

## Dev notes

The tooling (i.e. `env.sh` and related) is largely vibe-coded; so if it breaks or doesn't work for a given platform, just pull the slot machine handle until it works again (vibe code it some more).

The intention is to not use AI for any of the actual code (`AGENTS.md` should reflect this).

There's some one-time setup stuff you'll need:

- Docker and Docker Compose
- binfmt (possibly only if you're on an AMD64 platform)
  - `docker run --privileged --rm tonistiigi/binfmt --install all`
- Vagrant
- libvirt (Linux)
- socket_vmnet (macOS) — the qemu provider can't do the `192.168.56.x` private
  network on its own, and that's what the `pkg/deploy` tests target
  - `brew install socket_vmnet`
  - It has to run as root, in `shared` mode (`host` mode doesn't pass
    host-to-guest traffic), pinned to our gateway (Homebrew's default is
    `192.168.105.1`) with DHCP stopping below the static IPs:

```shell
sudo mkdir -p /opt/homebrew/var/log/socket_vmnet

sudo tee /Library/LaunchDaemons/io.github.lima-vm.socket_vmnet.plist >/dev/null <<'EOF'
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
	<dict>
		<key>Label</key>
		<string>io.github.lima-vm.socket_vmnet</string>
		<key>Program</key>
		<string>/opt/homebrew/opt/socket_vmnet/bin/socket_vmnet</string>
		<key>ProgramArguments</key>
		<array>
			<string>/opt/homebrew/opt/socket_vmnet/bin/socket_vmnet</string>
			<string>--vmnet-mode=shared</string>
			<string>--vmnet-gateway=192.168.56.1</string>
			<string>--vmnet-dhcp-end=192.168.56.10</string>
			<string>--vmnet-mask=255.255.255.0</string>
			<string>/opt/homebrew/var/run/socket_vmnet</string>
		</array>
		<key>StandardErrorPath</key>
		<string>/opt/homebrew/var/log/socket_vmnet/stderr</string>
		<key>StandardOutPath</key>
		<string>/opt/homebrew/var/log/socket_vmnet/stdout</string>
		<key>RunAtLoad</key>
		<true />
		<key>KeepAlive</key>
		<true />
		<key>UserName</key>
		<string>root</string>
		<key>ProcessType</key>
		<string>Interactive</string>
	</dict>
</plist>
EOF

sudo launchctl bootstrap system /Library/LaunchDaemons/io.github.lima-vm.socket_vmnet.plist
```

Dev workflow is something like this (manual control):

```shell
# bring up the containers and VMs and snapshot them
./env.sh up

# run the tests
./test.sh

# restore the snapshot
./env.sh restore
```

Or if you want a more automated approach:

```shell
# bring up the containers and VMs and snapshot them
./env.sh up

# watch for changes to any Go files, restore the snapshot and then run the tests
./test.sh watch-restore
```
