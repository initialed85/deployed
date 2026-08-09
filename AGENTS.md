# AGENTS.md

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
