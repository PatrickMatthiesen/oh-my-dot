#!/bin/bash
set -e

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# GitHub repository information
REPO_OWNER="PatrickMatthiesen"
REPO_NAME="oh-my-dot"

echo "Installing oh-my-dot..."

# Detect OS
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
case "$OS" in
    linux*)
        OS="linux"
        ;;
    darwin*)
        OS="darwin"
        ;;
    *)
        echo -e "${RED}Error: Unsupported operating system: $OS${NC}"
        echo "oh-my-dot supports Linux and macOS (darwin)."
        exit 1
        ;;
esac

# Detect architecture
ARCH=$(uname -m)
case "$ARCH" in
    x86_64|amd64)
        ARCH="x86_64"
        ;;
    arm64|aarch64)
        ARCH="arm64"
        ;;
    *)
        echo -e "${RED}Error: Unsupported architecture: $ARCH${NC}"
        echo "oh-my-dot supports x86_64 and arm64 architectures."
        exit 1
        ;;
esac

echo "Detected platform: $OS/$ARCH"

# Get version to install
if [ -n "$OH_MY_DOT_VERSION" ]; then
    TAG_NAME="$OH_MY_DOT_VERSION"
    echo "Installing specified version: $TAG_NAME"
else
    # Get latest release information from GitHub API
    echo "Fetching latest release information..."
    LATEST_RELEASE=$(curl -s "https://api.github.com/repos/$REPO_OWNER/$REPO_NAME/releases/latest")

    # Extract tag name
    TAG_NAME=$(echo "$LATEST_RELEASE" | grep '"tag_name":' | sed -E 's/.*"tag_name": "([^"]+)".*/\1/')

    if [ -z "$TAG_NAME" ]; then
        echo -e "${RED}Error: Failed to fetch latest release information${NC}"
        echo "You can specify a version manually by setting the OH_MY_DOT_VERSION environment variable:"
        echo "  OH_MY_DOT_VERSION=v0.0.25 bash install.sh"
        exit 1
    fi

    echo "Latest release: $TAG_NAME"
fi

# Construct download URL based on OS and architecture
ARCHIVE_NAME="${REPO_NAME}_${OS}_${ARCH}.tar.gz"
DOWNLOAD_URL="https://github.com/$REPO_OWNER/$REPO_NAME/releases/download/$TAG_NAME/$ARCHIVE_NAME"

echo "Downloading $ARCHIVE_NAME..."

# Create temporary directory for download
TMP_DIR=$(mktemp -d)
trap "rm -rf $TMP_DIR" EXIT

# Download the archive
if ! curl -fsSL "$DOWNLOAD_URL" -o "$TMP_DIR/$ARCHIVE_NAME"; then
    echo -e "${RED}Error: Failed to download $DOWNLOAD_URL${NC}"
    exit 1
fi

# Create installation directory
INSTALL_DIR="$HOME/.oh-my-dot/bin"
mkdir -p "$INSTALL_DIR"

# Extract the binary
echo "Extracting binary to $INSTALL_DIR..."
tar -xzf "$TMP_DIR/$ARCHIVE_NAME" -C "$TMP_DIR"

# Move the binary to the installation directory
mv "$TMP_DIR/oh-my-dot" "$INSTALL_DIR/oh-my-dot"
chmod +x "$INSTALL_DIR/oh-my-dot"

echo -e "${GREEN}✓ Binary installed to $INSTALL_DIR/oh-my-dot${NC}"

# Create symlink directory if it doesn't exist
SYMLINK_DIR="$HOME/.local/bin"
mkdir -p "$SYMLINK_DIR"

# Create or update symlink
SYMLINK_PATH="$SYMLINK_DIR/oh-my-dot"
if [ -L "$SYMLINK_PATH" ] || [ -f "$SYMLINK_PATH" ]; then
    rm -f "$SYMLINK_PATH"
fi

ln -s "$INSTALL_DIR/oh-my-dot" "$SYMLINK_PATH"
echo -e "${GREEN}✓ Created symlink: $SYMLINK_PATH -> $INSTALL_DIR/oh-my-dot${NC}"

# Add to current session's PATH
export PATH="$INSTALL_DIR:$PATH"
echo -e "${GREEN}✓ Added $INSTALL_DIR to current session's PATH${NC}"

# Check if ~/.local/bin is in the user's PATH
if [[ ":$PATH:" != *":$SYMLINK_DIR:"* ]]; then
    echo ""
    echo -e "${YELLOW}⚠ Warning: $SYMLINK_DIR is not in your PATH${NC}"
    echo ""
    echo "To use oh-my-dot, you need to add one of the following to your PATH:"
    echo "  1. Add $SYMLINK_DIR to your PATH (recommended)"
    echo "  2. Add $INSTALL_DIR to your PATH"
    echo ""
    echo "Add one of these lines to your shell's configuration file:"
    echo "  For bash (~/.bashrc or ~/.bash_profile):"
    echo "    export PATH=\"\$HOME/.local/bin:\$PATH\""
    echo "  For zsh (~/.zshrc):"
    echo "    export PATH=\"\$HOME/.local/bin:\$PATH\""
    echo "  For fish (~/.config/fish/config.fish):"
    echo "    set -gx PATH \$HOME/.local/bin \$PATH"
    echo ""
    echo "Or alternatively, use the installation directory directly:"
    echo "    export PATH=\"\$HOME/.oh-my-dot/bin:\$PATH\""
    echo ""
else
    echo ""
    echo -e "${GREEN}✓ Installation complete!${NC}"
    echo ""
fi

echo "You can now use oh-my-dot in this session by running: oh-my-dot"
echo "Run 'oh-my-dot --help' to get started."
