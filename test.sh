#!/bin/bash

set -e

if ! docker compose ps 2>&1 | grep Up >/dev/null 2>&1; then
	echo "error: doesn't look like the Docker Compose environment is up; have you run ./env.sh up?"
	exit 1
fi

if ! ./env.sh vagrant status 2>&1 | grep running >/dev/null 2>&1; then
	echo "error: doesn't look like the Vagrant environment is up; have you run ./env.sh up?"
	exit 1
fi

function test() {
	go test -v -count=1 -failfast ./pkg/ssh "${*}"
	echo ''

	go test -v -count=1 -failfast ./pkg/deploy "${*}"
	echo ''
}
export -f test

if [[ "${1}" == "watch" ]]; then
	shift
	find . -type f -name '*.go' | entr -n -cc -s "test ${*}"
else
	test "${*}"
fi
