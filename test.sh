#!/bin/bash

set -e

function test() {
	if [[ "${1}" == "" ]]; then
		go test -v -count=1 -failfast "$(for x in $(find . -type f -name '*_test.go' | sort); do dirname "${x}" | uniq; done | grep -vE 'local|ssh|rollout|deploy')"
		go test -v -count=1 -failfast "$(for x in $(find . -type f -name '*_test.go' | sort); do dirname "${x}" | uniq; done | grep local)"
		go test -v -count=1 -failfast "$(for x in $(find . -type f -name '*_test.go' | sort); do dirname "${x}" | uniq; done | grep ssh)"
		go test -v -count=1 -failfast "$(for x in $(find . -type f -name '*_test.go' | sort); do dirname "${x}" | uniq; done | grep deploy)"
		go test -v -count=1 -failfast "$(for x in $(find . -type f -name '*_test.go' | sort); do dirname "${x}" | uniq; done | grep rollout)"
	else
		go test -v -count=1 -failfast "${*}"
	fi
}
export -f test

function test_e2e() {
	go build -o ./deployed .

	./deployed rollout test/k3s.yaml
}
export -f test_e2e

if [[ "${1}" == "watch-restore" ]]; then
	shift

	if [[ "${1}" == "" ]]; then
		find go.mod main.go pkg test | entr -n -cc -s "go vet ./... && ./env.sh restore && test"
	else
		find go.mod main.go pkg test | entr -n -cc -s "go vet ./... && ./env.sh restore && test ${*}"
	fi
elif [[ "${1}" == "watch" ]]; then
	shift

	if [[ "${1}" == "" ]]; then
		find go.mod main.go pkg test | entr -n -cc -s "go vet ./... && test"
	else
		find go.mod main.go pkg test | entr -n -cc -s "go vet ./... && test ${*}"
	fi
elif [[ "${1}" == "restore" ]]; then
	shift

	go vet ./...

	./env.sh restore

	if [[ "${1}" == "" ]]; then
		test
	else
		test "${*}"
	fi
elif [[ "${1}" == "e2e-restore" ]]; then
	go vet ./...

	./env.sh restore

	test_e2e
elif [[ "${1}" == "e2e" ]]; then
	go vet ./...

	test_e2e
elif [[ "${1}" == "" ]]; then
	go vet ./...

	test
else
	go vet ./...

	test "${*}"
fi
