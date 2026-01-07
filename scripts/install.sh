#!/usr/bin/env sh
set -eu

ROOT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)

BIN_NAME=${BIN_NAME:-pitchstack}
PREFIX=${PREFIX:-}
OUT_DIR=${OUT_DIR:-}

default_out_dir() {
  os=$(uname -s 2>/dev/null || echo unknown)
  case "$os" in
    Darwin)
      # Homebrew on Apple Silicon is typically user-writable and on PATH.
      if [ -d /opt/homebrew/bin ] && [ -w /opt/homebrew/bin ]; then
        echo "/opt/homebrew/bin"
        return 0
      fi
      echo "/usr/local/bin"
      ;;
    Linux)
      echo "/usr/local/bin"
      ;;
    *)
      echo "$HOME/bin"
      ;;
  esac
}

if [ -z "$OUT_DIR" ]; then
  if [ -n "$PREFIX" ]; then
    OUT_DIR="$PREFIX/bin"
  else
    OUT_DIR="$(default_out_dir)"
  fi
fi

VERSION=${VERSION:-dev}
COMMIT=${COMMIT:-$(cd "$ROOT_DIR" && (git rev-parse --short HEAD 2>/dev/null || echo none))}

LDFLAGS="-s -w \
  -X github.com/pitchstack-gg/pitchstack-cli/internal/buildinfo.Version=$VERSION \
  -X github.com/pitchstack-gg/pitchstack-cli/internal/buildinfo.Commit=$COMMIT"

mkdir -p "$OUT_DIR"

if [ ! -w "$OUT_DIR" ]; then
  echo "error: $OUT_DIR is not writable." >&2
  echo "hint: re-run with sudo, or set OUT_DIR or PREFIX to a user-writable directory." >&2
  exit 1
fi

echo "Installing $BIN_NAME to $OUT_DIR/$BIN_NAME"
cd "$ROOT_DIR"
go build -ldflags "$LDFLAGS" -o "$OUT_DIR/$BIN_NAME" ./cmd/pitchstack

echo "Done."
echo "Make sure $OUT_DIR is on your PATH."
