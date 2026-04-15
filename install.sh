#!/bin/sh
set -e

# DevDash CLI installer
# Usage: curl -fsSL https://raw.githubusercontent.com/devdashproject/devdash-cli/main/install.sh | sh

REPO="devdashproject/devdash-cli"
INSTALL_DIR="${DEVDASH_INSTALL_DIR:-$HOME/.local/bin}"
BINARY="devdash"

# Detect platform
OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"

case "$ARCH" in
  x86_64|amd64) ARCH="amd64" ;;
  arm64|aarch64) ARCH="arm64" ;;
  *)
    echo "Error: unsupported architecture: $ARCH"
    exit 1
    ;;
esac

case "$OS" in
  darwin|linux) EXT="tar.gz" ;;
  mingw*|msys*|cygwin*|windows*) OS="windows"; EXT="zip" ;;
  *)
    echo "Error: unsupported OS: $OS"
    exit 1
    ;;
esac

# Get latest version
echo "Fetching latest version..."
VERSION=$(curl -fsSL "https://api.github.com/repos/$REPO/releases/latest" | grep '"tag_name"' | sed -E 's/.*"v([^"]+)".*/\1/')

if [ -z "$VERSION" ]; then
  echo "Error: could not determine latest version"
  exit 1
fi

echo "Installing devdash v${VERSION} (${OS}/${ARCH})..."

# Download
ARCHIVE="devdash_${VERSION}_${OS}_${ARCH}.${EXT}"
URL="https://github.com/$REPO/releases/download/v${VERSION}/${ARCHIVE}"

if [ "$OS" = "windows" ]; then
  BINARY="devdash.exe"
fi

TMPDIR=$(mktemp -d)
trap 'rm -rf "$TMPDIR"' EXIT

echo "Downloading $URL..."
curl -fsSL "$URL" -o "$TMPDIR/$ARCHIVE"

# Extract
if [ "$EXT" = "zip" ]; then
  unzip -q "$TMPDIR/$ARCHIVE" -d "$TMPDIR"
else
  tar -xzf "$TMPDIR/$ARCHIVE" -C "$TMPDIR"
fi

# Install
mkdir -p "$INSTALL_DIR"
cp "$TMPDIR/$BINARY" "$INSTALL_DIR/$BINARY"
chmod +x "$INSTALL_DIR/$BINARY"

echo "Installed to $INSTALL_DIR/$BINARY"

# Check PATH
case ":$PATH:" in
  *":$INSTALL_DIR:"*) ;;
  *)
    echo ""
    echo "Note: $INSTALL_DIR is not in your PATH."
    echo "Add it with:"
    echo "  export PATH=\"$INSTALL_DIR:\$PATH\""
    echo ""
    SHELL_NAME=$(basename "$SHELL")
    case "$SHELL_NAME" in
      zsh)  echo "  echo 'export PATH=\"$INSTALL_DIR:\$PATH\"' >> ~/.zshrc" ;;
      bash) echo "  echo 'export PATH=\"$INSTALL_DIR:\$PATH\"' >> ~/.bashrc" ;;
      fish) echo "  fish_add_path $INSTALL_DIR" ;;
    esac
    ;;
esac

echo ""
echo "Run 'devdash version' to verify."
