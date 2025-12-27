#!/bin/bash
#
# Fluxbase CLI Install Script
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/fluxbase-eu/fluxbase/main/install-cli.sh | bash
#   curl -fsSL https://raw.githubusercontent.com/fluxbase-eu/fluxbase/main/install-cli.sh | bash -s -- v0.1.0
#
# Environment variables:
#   FLUXBASE_INSTALL_DIR - Installation directory (default: /usr/local/bin)
#

set -e

REPO="fluxbase-eu/fluxbase"
INSTALL_DIR="${FLUXBASE_INSTALL_DIR:-/usr/local/bin}"
BINARY_NAME="fluxbase"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

error() {
    echo -e "${RED}[ERROR]${NC} $1"
    exit 1
}

# Detect OS and architecture
detect_platform() {
    OS=$(uname -s | tr '[:upper:]' '[:lower:]')
    ARCH=$(uname -m)

    case "$OS" in
        linux)
            OS="linux"
            ;;
        darwin)
            OS="darwin"
            ;;
        mingw*|msys*|cygwin*)
            OS="windows"
            ;;
        *)
            error "Unsupported operating system: $OS"
            ;;
    esac

    case "$ARCH" in
        x86_64|amd64)
            ARCH="amd64"
            ;;
        arm64|aarch64)
            ARCH="arm64"
            ;;
        *)
            error "Unsupported architecture: $ARCH"
            ;;
    esac

    PLATFORM="${OS}-${ARCH}"
    info "Detected platform: $PLATFORM"
}

# Get the latest version from GitHub releases
get_latest_version() {
    # First try to get the latest stable release
    LATEST=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" 2>/dev/null | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')

    # If no stable release exists, get the most recent release (including prereleases)
    if [ -z "$LATEST" ]; then
        LATEST=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases" | grep '"tag_name":' | head -1 | sed -E 's/.*"([^"]+)".*/\1/')
    fi

    if [ -z "$LATEST" ]; then
        error "Failed to fetch latest version"
    fi
    echo "$LATEST"
}

# Download and install the binary
install_binary() {
    VERSION="$1"

    # Construct download URL
    if [ "$OS" = "windows" ]; then
        FILENAME="${BINARY_NAME}-${PLATFORM}.zip"
    else
        FILENAME="${BINARY_NAME}-${PLATFORM}.tar.gz"
    fi

    DOWNLOAD_URL="https://github.com/${REPO}/releases/download/${VERSION}/${FILENAME}"

    info "Downloading ${BINARY_NAME} ${VERSION} for ${PLATFORM}..."

    # Create temp directory
    TMP_DIR=$(mktemp -d)
    trap "rm -rf $TMP_DIR" EXIT

    # Download
    if ! curl -fsSL "$DOWNLOAD_URL" -o "$TMP_DIR/$FILENAME"; then
        error "Failed to download from $DOWNLOAD_URL"
    fi

    # Extract
    info "Extracting..."
    cd "$TMP_DIR"
    if [ "$OS" = "windows" ]; then
        unzip -q "$FILENAME"
        EXTRACTED_BINARY="${BINARY_NAME}-${PLATFORM}.exe"
    else
        tar -xzf "$FILENAME"
        EXTRACTED_BINARY="${BINARY_NAME}-${PLATFORM}"
    fi

    # Install
    info "Installing to ${INSTALL_DIR}..."
    if [ ! -d "$INSTALL_DIR" ]; then
        mkdir -p "$INSTALL_DIR" 2>/dev/null || sudo mkdir -p "$INSTALL_DIR"
    fi

    if [ -w "$INSTALL_DIR" ]; then
        mv "$EXTRACTED_BINARY" "${INSTALL_DIR}/${BINARY_NAME}"
        chmod +x "${INSTALL_DIR}/${BINARY_NAME}"
    else
        sudo mv "$EXTRACTED_BINARY" "${INSTALL_DIR}/${BINARY_NAME}"
        sudo chmod +x "${INSTALL_DIR}/${BINARY_NAME}"
    fi

    info "Successfully installed ${BINARY_NAME} to ${INSTALL_DIR}/${BINARY_NAME}"
}

# Verify installation
verify_installation() {
    if command -v "$BINARY_NAME" &> /dev/null; then
        info "Verifying installation..."
        "$BINARY_NAME" version
        echo ""
        info "Installation complete! Run '${BINARY_NAME} --help' to get started."
    else
        warn "Installation complete, but '${BINARY_NAME}' is not in your PATH."
        warn "Add ${INSTALL_DIR} to your PATH, or run: ${INSTALL_DIR}/${BINARY_NAME}"
    fi
}

main() {
    info "Fluxbase CLI Installer"
    echo ""

    detect_platform

    # Use provided version or fetch latest
    if [ -n "$1" ]; then
        VERSION="$1"
        info "Installing version: $VERSION"
    else
        VERSION=$(get_latest_version)
        info "Latest version: $VERSION"
    fi

    install_binary "$VERSION"
    verify_installation
}

main "$@"
