#!/bin/bash

set -e

function do_test() {
	if [[ "${1}" == "" ]]; then
		go test -v -count=1 -failfast "$(for x in $(find . -type f -name '*_test.go' | sort); do dirname "${x}"; done | uniq | sort | grep -vE 'connection|rollout|deploy')"
		go test -v -count=1 -failfast "$(for x in $(find . -type f -name '*_test.go' | sort); do dirname "${x}"; done | uniq | sort | grep connection)"
		go test -v -count=1 -failfast "$(for x in $(find . -type f -name '*_test.go' | sort); do dirname "${x}"; done | uniq | sort | grep deploy)"
		go test -v -count=1 -failfast "$(for x in $(find . -type f -name '*_test.go' | sort); do dirname "${x}"; done | uniq | sort | grep rollout)"
	else
		go test -v -count=1 -failfast "${@}"
	fi

	echo -e "\ndone."
}
export -f do_test

function do_test_e2e() {
	go build -o ./deployed .

	./deployed rollout test/k3s.yaml

	echo -e "\ndone."
}
export -f do_test_e2e

if [[ "${1}" == "watch-restore" ]]; then
	shift

	if [[ "${1}" == "" ]]; then
		find go.mod main.go pkg test | entr -n -cc -s "go vet ./... && ./env.sh restore && do_test"
	else
		find go.mod main.go pkg test | entr -n -cc -s "go vet ./... && ./env.sh restore && do_test ${*}"
	fi
elif [[ "${1}" == "watch" ]]; then
	shift

	if [[ "${1}" == "" ]]; then
		find go.mod main.go pkg test | entr -n -cc -s "go vet ./... && do_test"
	else
		find go.mod main.go pkg test | entr -n -cc -s "go vet ./... && do_test ${*}"
	fi
elif [[ "${1}" == "e2e-watch" ]]; then
	shift

	find go.mod main.go pkg test | entr -n -cc -s "go vet ./... && do_test_e2e"
elif [[ "${1}" == "restore" ]]; then
	shift

	go vet ./...

	./env.sh restore

	if [[ "${1}" == "" ]]; then
		do_test
	else
		do_test "${@}"
	fi
elif [[ "${1}" == "e2e-restore" ]]; then
	go vet ./...

	./env.sh restore

	do_test_e2e
elif [[ "${1}" == "e2e" ]]; then
	go vet ./...

	do_test_e2e
elif [[ "${1}" == "" ]]; then
	go vet ./...

	do_test
else
	go vet ./...

	do_test "${@}"
fi
