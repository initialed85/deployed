#!/bin/bash

set -e

function _do_build() {
	echo -e "building...\n"

	jq <./schema/workspace.schema.json >./schema/workspace.schema.json.tmp
	mv -f ./schema/workspace.schema.json.tmp ./schema/workspace.schema.json

	go fmt .

	go generate .

	CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o ./deployed -trimpath .

	if [[ "${COMPRESS}" == "1" ]]; then
		if [[ "${GOOS}" == "darwin" ]]; then
			upx --best --lzma --force-macos ./deployed
		else
			upx --best --lzma ./deployed
		fi
	fi

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
