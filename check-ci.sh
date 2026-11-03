#!/bin/bash

# Runs the commands from .woodpecker/ci.yaml verbatim, in a clean golang:1.26 container, against a
# pristine copy of the working tree. Woodpecker concatenates a step's commands into one script, so
# this does the same; the list is read out of ci.yaml rather than duplicated here.
#
# ./env.sh up is not required: the tests that need the SSH fixture skip themselves, and the live
# infrastructure tests are skipped by name. What this catches is the lane itself - a missing system
# package, a gofmt violation, a test that only passes on a host with a lab.

set -e

image="${DEPLOYED_CI_IMAGE:-docker.io/library/golang:1.26}"
src="$(mktemp -d)"
container="deployed-check-ci-$$"

cleanup() {
	docker rm -f "$container" >/dev/null 2>&1 || true
	rm -rf "$src"
}
trap cleanup EXIT INT TERM

rsync -a --exclude .git --exclude .vagrant --exclude dist --exclude deployed ./ "$src/"

# pull the command list out of the test step, unquoting YAML single-quoted scalars on the way
awk '
BEGIN { q = sprintf("%c", 39) }
/^    commands:/ { in_list = 1; next }
in_list && /^      - / {
	line = substr($0, 9)
	if (line ~ ("^" q) && line ~ (q "$")) {
		line = substr(line, 2, length(line) - 2)
		gsub(q q, q, line)
	}
	print line
	next
}
in_list && /^      #/ { next }
in_list { in_list = 0 }
' .woodpecker/ci.yaml >"$src/ci-commands.sh"

if [[ ! -s "$src/ci-commands.sh" ]]; then
	echo "check-ci.sh: found no commands under the test step in .woodpecker/ci.yaml" >&2
	exit 1
fi

echo "check-ci.sh: lane commands, as the container will run them:"
sed 's/^/  | /' "$src/ci-commands.sh"

docker run -d --name "$container" -v "$src:/src" -w /src "$image" sleep 3600 >/dev/null

set +e
docker exec "$container" bash -e /src/ci-commands.sh
lane_exit="${?}"
set -e

if [[ "$lane_exit" == "0" ]]; then
	echo "check-ci.sh: lane passed (exit 0) in $image"
else
	echo "check-ci.sh: lane failed (exit $lane_exit) in $image" >&2
fi

exit "$lane_exit"
