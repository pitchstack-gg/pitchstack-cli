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

checksum_cmd() {
  if command -v sha256sum >/dev/null 2>&1; then
    echo sha256sum
    return 0
  fi
  if command -v shasum >/dev/null 2>&1; then
    echo "shasum -a 256"
    return 0
  fi
  echo "error: missing required command: sha256sum or shasum" >&2
  exit 1
}

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

release_json=$(curl -fsSL "https://api.github.com/repos/$REPO/releases/latest")

asset_url=$(
  printf '%s\n' "$release_json" |
    sed -n 's/.*"browser_download_url": "\(.*\)".*/\1/p' |
    grep "_${os}_${arch}\.tar\.gz$" |
    head -n 1
)
checksums_url=$(
  printf '%s\n' "$release_json" |
    sed -n 's/.*"browser_download_url": "\(.*checksums\.txt\)".*/\1/p' |
    head -n 1
)

if [ -z "$asset_url" ]; then
  echo "error: could not find latest release asset for ${os}_${arch}" >&2
  exit 1
fi
if [ -z "$checksums_url" ]; then
  echo "error: could not find latest release checksums.txt" >&2
  exit 1
fi

echo "Downloading $asset_url"
archive="$tmp/$BIN_NAME.tar.gz"
curl -fsSL "$asset_url" -o "$archive"
curl -fsSL "$checksums_url" -o "$tmp/checksums.txt"

archive_name=$(basename "$asset_url")
expected=$(grep "[[:space:]]$archive_name\$" "$tmp/checksums.txt" | awk '{print $1}' | head -n 1)
if [ -z "$expected" ]; then
  echo "error: checksum not found for $archive_name" >&2
  exit 1
fi
actual=$($(checksum_cmd) "$archive" | awk '{print $1}')
if [ "$actual" != "$expected" ]; then
  echo "error: checksum mismatch for $archive_name" >&2
  exit 1
fi

tar -xzf "$archive" -C "$tmp"

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
