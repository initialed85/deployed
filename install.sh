#!/bin/bash

set -e

function _do_install() {
	echo -e "building...\n"
	
	go install -trimpath .

	find "$(realpath "$(command -v deployed)")"
	ls -al "$(realpath "$(command -v deployed)")"
	du -sh "$(realpath "$(command -v deployed)")"
	
	echo -e "\ndone."
}
export -f _do_install

function do_install() {
	time _do_install
	echo ""
}
export -f do_install

if [[ "${1}" == "watch" ]]; then
	find test.sh build.sh install.sh go.mod main.go pkg test | entr -n -cc -s "do_install"
else
	do_install
fi

go install -x -trimpath .
