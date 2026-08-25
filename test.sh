#!/bin/bash

set -e

# ignore warning that this function isn't invoked- it is, in a built command string
# shellcheck disable=SC2329
function do_test_e2e() {
	CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o ./deployed -trimpath .

	./deployed rollout test/k3s.yaml

	echo -e "\ndone."
}
export -f do_test_e2e

# ignore warning that this function isn't invoked- it is, in a built command string
# shellcheck disable=SC2329
function do_test() {
	if [[ "${#}" == "0" ]]; then
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

do_restore="0"
do_watch="0"
do_test_e2e="0"
do_test="0"

case "${1:-}" in

restore)
	do_restore="1"
	do_test="1"
	shift
	;;

watch)
	do_watch="1"
	do_test="1"
	shift
	;;

watch-restore)
	do_test="1"
	do_watch="1"
	do_restore="1"
	shift
	;;

e2e-restore)
	do_test_e2e="1"
	do_restore="1"
	shift
	;;

e2e-watch)
	do_test_e2e="1"
	do_watch="1"
	shift
	;;

e2e-watch-restore)
	do_test_e2e="1"
	do_watch="1"
	do_restore="1"
	shift
	;;

e2e)
	do_test_e2e="1"
	shift
	;;

*)
	do_test="1"
	;;

esac

# command="go vet ./... && staticcheck ./..."
command=""

if [[ "${do_restore}" == "1" ]]; then
	command="./env.sh restore"
fi

args=""
for arg in "${@}"; do
	printf -v quoted_arg "%q" "${arg}"
	args="${args} ${quoted_arg}"
done

if [[ "${do_test_e2e}" == "1" ]]; then
	command="${command}${command:+ && }do_test_e2e${args}"
elif [[ "${do_test}" == "1" ]]; then
	command="${command}${command:+ && }do_test${args}"
fi

command="set -e; ${command}"

if [[ "${do_watch}" == "1" ]]; then
	find go.mod main.go pkg test | entr -n -cc -s "${command}"
else
	bash -c "${command}"
fi

exit 0
