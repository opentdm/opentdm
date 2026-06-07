#!/usr/bin/env bash
# Install the opentdm CLI from GitHub Releases.
#   curl -fsSL https://raw.githubusercontent.com/opentdm/opentdm/main/install.sh | bash
#   ... | bash -s -- --version v0.2.0 --bin-dir /usr/local/bin
set -euo pipefail

REPO="opentdm/opentdm"
VERSION="latest"
BIN_DIR="${OPENTDM_BIN_DIR:-}"

while [ $# -gt 0 ]; do
  case "$1" in
    --version) VERSION="$2"; shift 2 ;;
    --bin-dir) BIN_DIR="$2"; shift 2 ;;
    *) echo "unknown arg: $1" >&2; exit 2 ;;
  esac
done

os="$(uname -s | tr '[:upper:]' '[:lower:]')"
arch="$(uname -m)"
case "$arch" in
  x86_64|amd64) arch="amd64" ;;
  aarch64|arm64) arch="arm64" ;;
  *) echo "unsupported arch: $arch" >&2; exit 1 ;;
esac

if [ "$VERSION" = "latest" ]; then
  VERSION="$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" | grep -m1 '"tag_name"' | cut -d'"' -f4)"
fi
[ -n "$VERSION" ] || { echo "could not resolve version" >&2; exit 1; }

asset="opentdm_${VERSION#v}_${os}_${arch}.tar.gz"
url="https://github.com/${REPO}/releases/download/${VERSION}/${asset}"

tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT
echo "Downloading $url" >&2
curl -fsSL "$url" -o "$tmp/opentdm.tar.gz"
# Verify checksum when available — abort on a real mismatch; warn only if no tool/entry exists.
# We compare hash strings directly (the download is saved as opentdm.tar.gz, not its asset name,
# so `sha256sum -c` against checksums.txt would never find the file).
if curl -fsSL "https://github.com/${REPO}/releases/download/${VERSION}/checksums.txt" -o "$tmp/checksums.txt" 2>/dev/null; then
  expected="$(grep " ${asset}\$" "$tmp/checksums.txt" | awk '{print $1}' || true)"
  if command -v sha256sum >/dev/null 2>&1; then
    actual="$(sha256sum "$tmp/opentdm.tar.gz" | awk '{print $1}')"
  elif command -v shasum >/dev/null 2>&1; then
    actual="$(shasum -a 256 "$tmp/opentdm.tar.gz" | awk '{print $1}')"   # macOS/BSD ship shasum, not sha256sum
  else
    actual=""
  fi
  if [ -z "$actual" ]; then
    echo "warning: no sha256 tool (sha256sum/shasum) found; checksum not verified" >&2
  elif [ -z "$expected" ]; then
    echo "warning: no checksum entry for ${asset}; not verified" >&2
  elif [ "$expected" = "$actual" ]; then
    echo "Checksum OK for ${asset}" >&2
  else
    echo "error: checksum verification FAILED for ${asset} (expected ${expected}, got ${actual})" >&2
    exit 1
  fi
fi
tar -xzf "$tmp/opentdm.tar.gz" -C "$tmp"

if [ -z "$BIN_DIR" ]; then
  if [ -w /usr/local/bin ]; then BIN_DIR=/usr/local/bin; else BIN_DIR="$HOME/.local/bin"; fi
fi
mkdir -p "$BIN_DIR"
install -m 0755 "$tmp/opentdm" "$BIN_DIR/opentdm"
echo "Installed opentdm $VERSION to $BIN_DIR/opentdm" >&2
