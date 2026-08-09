# deployed

## Goal

A simple way to provision software on Linux servers, via SSH.

## Concepts

- Target
  - A machine to provision (via SSH)
- Step
  - A script to run (via SSH) or some files to transfer (via SCP)
- Deployment
  - A collection of Targets and Steps

## Plan of attack

- [DONE] Build SSH layer
  - [DONE] Be able to run commands
  - [DONE] Be able to run commands with sudo
  - [DONE] Be able to transfer files via SCP
  - [DONE] Be able to transfer files via SCP and move them to a privileged location with sudo
  - [TODO] Expose SSH layer via an app entrypoint (as a low-level escape hatch)
- [WIP] Build deployment layer
  - [DONE] Be able to run a number of Steps against a Target
  - [WIP] Store a hash of the last run Steps on a Target for idempotency
  - [TODO] Expose deployment layer via an app entrypoint

## Dev notes

The tooling (e.g. `env.sh` and related) is largely vibe-coded; so if it breaks or doesn't work for a given platform, just give the dice a roll, vibe code it some more until it works.

The intention is to not use AI for any of the actual code (`AGENTS.md` should reflect this).

There's some one-time setup stuff you'll need:

- Docker and Docker Compose
- binfmt (possibly only if you're on an AMD64 platform)
  - `docker run --privileged --rm tonistiigi/binfmt --install all`
- Vagrant
- libvirt

```shell
./env.sh up

./test.sh watch
```

You can also snapshot and restore the VMs and containers like this:

```shell
./env.sh snapshot

./env.sh restore
```

By default `./env.sh up` will take a snapshot once the environment settles, so you can in theory call `./env.sh restore` at any point after starting to reset your environment.
