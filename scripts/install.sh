#!/bin/sh
# sdd universal installer for macOS and Linux
# Usage: curl -fsSL https://raw.githubusercontent.com/vikashkumar21/sdd/main/scripts/install.sh | sh

set -eu

# Detect OS
OS="$(uname -s)"
case "$OS" in
  Darwin) OS="darwin" ;;
  Linux)  OS="linux" ;;
  *)
    echo "Error: unsupported operating system: $OS" >&2
    echo "sdd installer currently supports macOS (Darwin) and Linux." >&2
    exit 1
    ;;
esac

# Detect Architecture
ARCH="$(uname -m)"
case "$ARCH" in
  x86_64|amd64)  ARCH="amd64" ;;
  arm64|aarch64) ARCH="arm64" ;;
  *)
    echo "Error: unsupported machine architecture: $ARCH" >&2
    echo "sdd installer currently supports x86_64/amd64 and arm64/aarch64." >&2
    exit 1
    ;;
esac

# Check HTTP downloader
if command -v curl >/dev/null 2>&1; then
  HTTP_GET="curl -fsSL"
elif command -v wget >/dev/null 2>&1; then
  HTTP_GET="wget -qO-"
else
  echo "Error: neither curl nor wget was found. Please install curl or wget." >&2
  exit 1
fi

REPO="${SDD_REPO:-vikashkumar21/sdd}"
VERSION="${SDD_VERSION:-}"
DRY_RUN="${SDD_DRY_RUN:-0}"
ASSET="sdd_${OS}_${ARCH}"
SUMS="SHA256SUMS"

if [ -n "${SDD_BASE_URL:-}" ]; then
  DOWNLOAD_URL="${SDD_BASE_URL}"
elif [ -n "$VERSION" ]; then
  DOWNLOAD_URL="https://github.com/${REPO}/releases/download/${VERSION}"
else
  DOWNLOAD_URL="https://github.com/${REPO}/releases/latest/download"
fi

# Prepare temporary directory
TMP_DIR="$(mktemp -d 2>/dev/null || mktemp -d -t 'sdd-install')"
cleanup() {
  rm -rf "$TMP_DIR"
}
trap cleanup EXIT INT TERM

BIN_PATH="${TMP_DIR}/${ASSET}"
SUMS_PATH="${TMP_DIR}/${SUMS}"

echo "Downloading sdd (${OS}/${ARCH})..."
$HTTP_GET "${DOWNLOAD_URL}/${ASSET}" > "$BIN_PATH"
chmod +x "$BIN_PATH"

# Verify SHA256 checksum if available
if $HTTP_GET "${DOWNLOAD_URL}/${SUMS}" > "$SUMS_PATH" 2>/dev/null; then
  EXPECTED_HASH="$(grep "dist/${ASSET}$" "$SUMS_PATH" 2>/dev/null | awk '{print $1}')"
  if [ -z "$EXPECTED_HASH" ]; then
    EXPECTED_HASH="$(grep "${ASSET}$" "$SUMS_PATH" 2>/dev/null | awk '{print $1}')"
  fi

  if [ -n "$EXPECTED_HASH" ]; then
    ACTUAL_HASH=""
    if command -v sha256sum >/dev/null 2>&1; then
      ACTUAL_HASH="$(sha256sum "$BIN_PATH" | awk '{print $1}')"
    elif command -v shasum >/dev/null 2>&1; then
      ACTUAL_HASH="$(shasum -a 256 "$BIN_PATH" | awk '{print $1}')"
    fi

    if [ -n "$ACTUAL_HASH" ]; then
      if [ "$ACTUAL_HASH" != "$EXPECTED_HASH" ]; then
        echo "Error: SHA256 checksum mismatch for ${ASSET}" >&2
        echo "Expected: ${EXPECTED_HASH}" >&2
        echo "Actual:   ${ACTUAL_HASH}" >&2
        exit 1
      fi
      echo "✓ Verified SHA256 checksum."
    fi
  fi
fi

# Remove macOS quarantine if present
if [ "$OS" = "darwin" ] && command -v xattr >/dev/null 2>&1; then
  xattr -d com.apple.quarantine "$BIN_PATH" 2>/dev/null || true
fi

# Delegate installation to sdd install
if [ "$DRY_RUN" = "1" ] || [ "$DRY_RUN" = "true" ]; then
  "$BIN_PATH" install --dry-run
else
  "$BIN_PATH" install
fi
