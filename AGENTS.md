# AGENTS.md

The goal of this project is for Ed (initialed85) to get back to enjoying tradcoding (writing code by hand).

So why do we need this tool? Well, Ed's take:

- Packer is good, but it's slow to iterate with- best left for the things we don't change often
- Terraform is good for instatiating Packer-baked VMs and not much else; cloud-init is a tedious way to manage what's installed
- That leaves us with Ansible... I just really don't like Ansible

So we're making our own.

## Scope

Tooling only: `env.sh`, `test.sh`, `Vagrantfile`, `docker-compose.yaml`, Dockerfiles, `.gitignore`. Do not edit Go source, tests, or application code — the user owns that.

## Style

Tight solutions. No bloat, no comments unless asked or the solution is confusing, no defensive code for cases that don't exist yet. Match the existing style of the file you're editing.

## Git discipline

Never `git stash` the working tree. The user is almost certainly editing files in parallel while we work. If we need a clean tree to test something, ask first or work around it (copy files, use a scratch dir). Never `git add`, `commit`, `push`, or `stash drop` without explicit instruction.

## Verification

After editing tooling, syntax-check (`bash -n`) and run the relevant command to confirm it works. Don't claim success without testing.

## Communication

Concise. Say what changed and why, nothing else. No preamble, no summary of work already shown.
