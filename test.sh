#!/bin/bash

set -e

function test() {
	go test -v -count=1 -failfast "${*}"
}
export -f test

if [[ "${1}" == "watch-restore" ]]; then
	shift

	if [[ "${1}" == "" ]]; then
		find main.go pkg test | entr -n -cc -s "go vet ./... && ./env.sh restore && test ./..."
	else
		find main.go pkg test | entr -n -cc -s "go vet ./... && ./env.sh restore && test ${*}"
	fi
	exit 0
fi

if [[ "${1}" == "watch" ]]; then
	shift

	if [[ "${1}" == "" ]]; then
		find main.go pkg test | entr -n -cc -s "go vet ./... && test ./..."
	else
		find main.go pkg test | entr -n -cc -s "go vet ./... && test ${*}"
	fi
	exit 0
fi

if [[ "${1}" == "restore" ]]; then
	shift

	go vet ./...

	./env.sh restore

	if [[ "${1}" == "" ]]; then
		test "./..."
	else
		test "${*}"
	fi
	exit 0
fi

if [[ "${1}" == "" ]]; then
	go vet ./...

	test "./..."
else
	go vet ./...

	test "${*}"
fi
