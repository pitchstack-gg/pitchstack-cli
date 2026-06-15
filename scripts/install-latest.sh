#!/usr/bin/env sh
set -eu

REPO=${REPO:-pitchstack-gg/pitchstack-cli}
BIN_NAME=${BIN_NAME:-pitchstack}
INSTALL_DIR=${INSTALL_DIR:-/usr/local/bin}

need() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "error: missing required command: $1" >&2
    exit 1
  fi
}

need curl
need tar

os=$(uname -s 2>/dev/null | tr '[:upper:]' '[:lower:]')
arch=$(uname -m 2>/dev/null)

case "$os" in
  darwin|linux) ;;
  *)
    echo "error: unsupported OS: $os" >&2
    exit 1
    ;;
esac

case "$arch" in
  x86_64|amd64) arch=amd64 ;;
  arm64|aarch64) arch=arm64 ;;
  *)
    echo "error: unsupported architecture: $arch" >&2
    exit 1
    ;;
esac

tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT INT TERM

asset_url=$(
  curl -fsSL "https://api.github.com/repos/$REPO/releases/latest" |
    sed -n 's/.*"browser_download_url": "\(.*\)".*/\1/p' |
    grep "_${os}_${arch}\.tar\.gz$" |
    head -n 1
)

if [ -z "$asset_url" ]; then
  echo "error: could not find latest release asset for ${os}_${arch}" >&2
  exit 1
fi

echo "Downloading $asset_url"
curl -fsSL "$asset_url" -o "$tmp/$BIN_NAME.tar.gz"
tar -xzf "$tmp/$BIN_NAME.tar.gz" -C "$tmp"

if [ ! -d "$INSTALL_DIR" ]; then
  if [ -w "$(dirname "$INSTALL_DIR")" ]; then
    mkdir -p "$INSTALL_DIR"
  else
    sudo mkdir -p "$INSTALL_DIR"
  fi
fi

if [ -w "$INSTALL_DIR" ]; then
  install -m 0755 "$tmp/$BIN_NAME" "$INSTALL_DIR/$BIN_NAME"
else
  sudo install -m 0755 "$tmp/$BIN_NAME" "$INSTALL_DIR/$BIN_NAME"
fi

echo "Installed $BIN_NAME to $INSTALL_DIR/$BIN_NAME"
"$INSTALL_DIR/$BIN_NAME" version
