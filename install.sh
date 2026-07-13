#!/usr/bin/env bash
# Install the latest hyperstudy-agent release binary into /usr/local/bin.
#
#   curl -fsSL https://raw.githubusercontent.com/hyperstudyio/hyperstudy-agent-demo/main/install.sh | bash
#
set -euo pipefail

REPO="hyperstudyio/hyperstudy-agent-demo"
BIN_NAME="hyperstudy-agent"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"

os="$(uname -s)"
arch="$(uname -m)"

case "$os" in
  Darwin) goos="darwin" ;;
  Linux)  goos="linux" ;;
  *)
    echo "error: unsupported OS: $os (only Darwin and Linux are published)" >&2
    exit 1
    ;;
esac

case "$arch" in
  arm64|aarch64) goarch="arm64" ;;
  x86_64|amd64)  goarch="amd64" ;;
  *)
    echo "error: unsupported architecture: $arch (only arm64 and amd64 are published)" >&2
    exit 1
    ;;
esac

if [ "$goos" = "darwin" ] && [ "$goarch" = "amd64" ]; then
  echo "error: darwin/amd64 is not published (Apple Silicon only). Build from source with 'go build' instead." >&2
  exit 1
fi

echo "Fetching latest release for $goos/$goarch..."

latest_url="https://api.github.com/repos/${REPO}/releases/latest"
tag="$(curl -fsSL "$latest_url" | grep '"tag_name":' | head -1 | sed -E 's/.*"tag_name": *"([^"]+)".*/\1/')"

if [ -z "$tag" ]; then
  echo "error: could not determine the latest release tag from $latest_url" >&2
  exit 1
fi

version="${tag#v}"
archive="${BIN_NAME}_${version}_${goos}_${goarch}.tar.gz"
download_url="https://github.com/${REPO}/releases/download/${tag}/${archive}"

tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"' EXIT

echo "Downloading ${download_url}..."
curl -fsSL "$download_url" -o "$tmpdir/$archive"

tar -xzf "$tmpdir/$archive" -C "$tmpdir" "$BIN_NAME"

if [ -w "$INSTALL_DIR" ]; then
  mv "$tmpdir/$BIN_NAME" "$INSTALL_DIR/$BIN_NAME"
else
  echo "Installing to $INSTALL_DIR requires sudo:"
  sudo mv "$tmpdir/$BIN_NAME" "$INSTALL_DIR/$BIN_NAME"
fi
chmod +x "$INSTALL_DIR/$BIN_NAME"

echo "Installed ${tag} to ${INSTALL_DIR}/${BIN_NAME}"
"$INSTALL_DIR/$BIN_NAME" --version
