#!/bin/bash

set -e

function _do_build() {
	echo -e "building...\n"

	go generate .

	go build -o ./deployed -trimpath .

	find "$(realpath ./deployed)"
	ls -al "$(realpath ./deployed)"
	du -sh "$(realpath ./deployed)"

	echo -e "\ndone."
}
export -f _do_build

function do_build() {
	time _do_build
	echo ""
}
export -f do_build

if [[ "${1}" == "watch" ]]; then
	find test.sh build.sh install.sh go.mod main.go pkg test | entr -n -cc -s "do_build"
else
	do_build
fi
