#!/bin/sh

set -eu

repository=${DEPLOYED_REPOSITORY:-initialed85/deployed}
version=${DEPLOYED_VERSION:-latest}

case "$(uname -s)" in
Linux)
	goos=linux
	;;
Darwin)
	goos=darwin
	;;
*)
	echo "unsupported operating system: $(uname -s)" >&2
	exit 1
	;;
esac

case "$(uname -m)" in
x86_64 | amd64)
	goarch=amd64
	;;
aarch64 | arm64)
	goarch=arm64
	;;
*)
	echo "unsupported architecture: $(uname -m)" >&2
	exit 1
	;;
esac

case "$goos/$goarch" in
linux/amd64 | linux/arm64 | darwin/arm64)
	;;
*)
	echo "unsupported platform: $goos/$goarch" >&2
	exit 1
	;;
esac

if [ -n "${DEPLOYED_INSTALL_DIR:-}" ]; then
	install_dir=$DEPLOYED_INSTALL_DIR
elif [ -n "${HOME:-}" ]; then
	install_dir=$HOME/.local/bin
else
	echo "HOME is not set; set DEPLOYED_INSTALL_DIR and try again" >&2
	exit 1
fi

asset="deployed-${goos}-${goarch}.tar.gz"
if [ "$version" = latest ]; then
	release_path=latest/download
else
	release_path="download/$version"
fi
base_url="https://github.com/$repository/releases/$release_path"
tmp_dir=$(mktemp -d)
trap 'rm -rf "$tmp_dir"' EXIT INT TERM

echo "downloading deployed ($goos/$goarch)..."
curl -fsSL "$base_url/$asset" -o "$tmp_dir/$asset"
curl -fsSL "$base_url/$asset.sha256" -o "$tmp_dir/$asset.sha256"

if command -v sha256sum >/dev/null 2>&1; then
	(cd "$tmp_dir" && sha256sum -c "$asset.sha256")
elif command -v shasum >/dev/null 2>&1; then
	expected=$(awk '{print $1}' "$tmp_dir/$asset.sha256")
	actual=$(shasum -a 256 "$tmp_dir/$asset" | awk '{print $1}')
	if [ "$expected" != "$actual" ]; then
		echo "checksum verification failed" >&2
		exit 1
	fi
else
	echo "sha256sum or shasum is required" >&2
	exit 1
fi

tar -xzf "$tmp_dir/$asset" -C "$tmp_dir"
mkdir -p "$install_dir"
install -m 0755 "$tmp_dir/deployed-$goos-$goarch" "$install_dir/deployed"

echo "installed deployed to $install_dir/deployed"
case ":${PATH:-}:" in
*":$install_dir:"*) ;;
*) echo "add $install_dir to your PATH" ;;
esac
