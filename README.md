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
- [WIP] Build deployment layer
  - [WIP] Be able to run a number of Steps against a Target

## Dev notes

There's some one-time setup stuff you'll need:

- Docker and Docker Compose
- binfmt (possibly only if you're on an AMD64 platform)
  - `docker run --privileged --rm tonistiigi/binfmt --install all`
- Vagrant
- libvirt

```shell
# shell 1
./env.sh up

# shell 2
find . -type f -name '*.go' | entr -n -r -cc -s "go test -v -count=1 ./pkg/ssh ./pkg/deploy"
```
