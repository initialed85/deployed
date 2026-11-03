# AGENTS.md

The goal of this project is for Ed (initialed85) to get back to enjoying tradcoding (writing code by hand).

So why do we need this tool? Well, Ed's take:

- Packer is good, but it's slow to iterate with- best left for the things we don't change often
- Terraform is good for instatiating Packer-baked VMs and not much else; cloud-init is a tedious way to manage what's installed
- That leaves us with Ansible... I just really don't like Ansible

So we're making our own.

## Scope

- Push back when asked to edit the actual implementation code; remind Ed the whole purpose of this thing is for him to do some tradcoding
- You might be asked to edit the tests, this is okay- don't just edit them freely though
- Feel free to edit the tooling (e.g. `env.sh`, `test.sh`, `Vagrantfile`, `docker-compose.yaml` and similar) but ensure to be thinking of both Linux and macOS compatibility

## Style

- Tight solutions.
- No bloat, no comments unless asked or the solution is confusing, no defensive code for cases that don't exist yet.
- Match the existing style of the file you're editing.

## Git discipline

- Avoid `git stash` on the working tree, Ed is probably working alongside you.
- If we need a clean tree to test something, ask first or work around it (copy files, use a scratch dir).
- Never `git add`, `commit`, `push`, or `stash drop` without explicit instruction.

## Test tiers

`go test ./...` is not one suite, and reading a failure without knowing the tier sends you the wrong way:

- Unit tests need nothing. These are what CI actually runs.
- `TestSSH`, `TestDeployModes`, `TestRolloutModes` and `TestRolloutFolderMerge` dial `localhost:2221`,
  the `ubuntu-ssh-server` container `./env.sh up` builds from `Dockerfile.test`. That image is not
  published, and the fixture cannot live in the container that runs the tests (a step script is uploaded
  to the path it already occupies locally, and folder uploads are staged through `/tmp`), so these skip
  themselves through `pkg/helpers/fixture.RequireSSH` when nothing is listening.
- Serving that fixture from `services:` is not a shortcut either: our agents run on the Kubernetes
  backend, where a service is a separate pod reachable as `ubuntu-ssh-server:2221` and never as
  `localhost:2221`.
- `TestDeploy` and `TestRollout` deploy to the real Vagrant VMs on `192.168.56.11`-`14` with hardcoded
  credentials. They are a side effect, not a slow test, and the fmt/vet/test lane is an unprivileged
  container that cannot hold that subnet, so it skips them by name, anchored (`TestDeploy$`,
  `TestRollout$`), and the unit tests in the same files still run. They are not impossible in CI: a
  privileged vagrant-libvirt agent brings the same four VMs up inside a step pod (`.woodpecker/e2e.yaml`),
  so that skip list describes this lane, not a permanent rule about those tests.
- The lab serves its own k3s. `./env.sh up` builds and starts `tools/mirror` on `192.168.56.1:8080`,
  `.woodpecker/e2e.yaml` starts the same binary in the step pod, and the guest install scripts probe it and
  fall back to GitHub when nothing answers. Four guests pulling 81 MB from GitHub at the same instant is what
  made the live lane red; do not reintroduce that by hardcoding `get.k3s.io` into a spec.

## Verification

- After editing tooling, syntax-check (`bash -n`) and run the relevant command to confirm it works.
- Don't claim success without testing.
- Testing is best done via `./test.sh` as it has a `restore` concept baked in that can restore the VMs and containers to snapshot taken when the environment was started
- Before pushing anything the CI lane covers, run `./check-ci.sh`: it reads the commands out of `.woodpecker/ci.yaml`, runs them verbatim in a clean `golang:1.26` container, and reports the exit code it measured. A lane that only passes on a machine holding a lab is not a lane.
- Measure exit codes instead of reading the last line of a pipeline; several failures here look green when piped.

## Communication

Concise. Say what changed and why, nothing else. No preamble, no summary of work already shown.
