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

```shell
# shell 1
docker compose up --build ; docker compose down --volumes --remove-orphans

# shell 2
limactl create --cpus=1 --memory=2 --name ubuntu-1 --ssh-port 12221 --tty=false
limactl start ubuntu-1
```
