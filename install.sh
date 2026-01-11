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
    
    # Try with timeout and check for errors
    if ! LATEST_RELEASE=$(curl -sSf --max-time 10 "https://api.github.com/repos/$REPO_OWNER/$REPO_NAME/releases/latest" 2>&1); then
        echo -e "${RED}Error: Failed to fetch latest release information${NC}"
        echo "Network error or GitHub API unavailable."
        echo "You can specify a version manually by setting the OH_MY_DOT_VERSION environment variable:"
        echo "  OH_MY_DOT_VERSION=v0.0.25 bash install.sh"
        exit 1
    fi

    # Extract tag name - try jq first, fall back to grep/sed
    if command -v jq > /dev/null 2>&1; then
        TAG_NAME=$(echo "$LATEST_RELEASE" | jq -r '.tag_name')
    else
        # Try grep with PCRE if available (more readable)
        TAG_NAME=$(echo "$LATEST_RELEASE" | grep -Po '"tag_name":\s*"\K[^"]*' 2>/dev/null | head -1)
        
        # Fall back to basic grep/sed for maximum compatibility
        if [ -z "$TAG_NAME" ]; then
            TAG_NAME=$(echo "$LATEST_RELEASE" | grep -o '"tag_name"[[:space:]]*:[[:space:]]*"[^"]*"' | head -1 | sed -n 's/.*"tag_name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p')
        fi
    fi

    if [ -z "$TAG_NAME" ] || [ "$TAG_NAME" = "null" ]; then
        echo -e "${RED}Error: Failed to parse release information${NC}"
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
trap 'rm -rf "$TMP_DIR"' EXIT

# Download the archive with timeout and proper error handling
if ! curl -fsSL --max-time 60 "$DOWNLOAD_URL" -o "$TMP_DIR/$ARCHIVE_NAME"; then
    echo -e "${RED}Error: Failed to download $DOWNLOAD_URL${NC}"
    echo "Please check:"
    echo "  - Your internet connection"
    echo "  - The release exists: https://github.com/$REPO_OWNER/$REPO_NAME/releases/tag/$TAG_NAME"
    echo "  - The archive name is correct: $ARCHIVE_NAME"
    exit 1
fi

# Create installation directory
INSTALL_DIR="$HOME/.oh-my-dot/bin"
mkdir -p "$INSTALL_DIR"

# Extract the binary
echo "Extracting binary to $INSTALL_DIR..."
tar -xzf "$TMP_DIR/$ARCHIVE_NAME" -C "$TMP_DIR"

# Verify the binary exists
if [ ! -f "$TMP_DIR/oh-my-dot" ]; then
    echo -e "${RED}Error: Binary 'oh-my-dot' not found in archive${NC}"
    echo "Archive contents:"
    tar -tzf "$TMP_DIR/$ARCHIVE_NAME"
    exit 1
fi

# Move the binary to the installation directory
mv "$TMP_DIR/oh-my-dot" "$INSTALL_DIR/oh-my-dot"
chmod +x "$INSTALL_DIR/oh-my-dot"

echo -e "${GREEN}✓ Binary installed to $INSTALL_DIR/oh-my-dot${NC}"

# Create symlink directory if it doesn't exist
SYMLINK_DIR="$HOME/.local/bin"
mkdir -p "$SYMLINK_DIR"

# Create or update symlink
SYMLINK_PATH="$SYMLINK_DIR/oh-my-dot"
if [ -d "$SYMLINK_PATH" ]; then
    echo -e "${RED}Error: $SYMLINK_PATH is a directory${NC}"
    echo "Please remove or rename it before installing oh-my-dot"
    exit 1
elif [ -e "$SYMLINK_PATH" ] || [ -L "$SYMLINK_PATH" ]; then
    rm -f "$SYMLINK_PATH"
fi

ln -s "$INSTALL_DIR/oh-my-dot" "$SYMLINK_PATH"
echo -e "${GREEN}✓ Created symlink: $SYMLINK_PATH -> $INSTALL_DIR/oh-my-dot${NC}"

# Check if ~/.local/bin is in the user's PATH (before we modify it)
PATH_WARNING=false
case ":$PATH:" in
    *":$SYMLINK_DIR:"*)
        # ~/.local/bin is in PATH
        ;;
    *)
        PATH_WARNING=true
        ;;
esac

# Add to current session's PATH - use SYMLINK_DIR if it's already in PATH, otherwise use INSTALL_DIR
if [ "$PATH_WARNING" = false ]; then
    # ~/.local/bin is already in PATH, so the symlink will work
    echo -e "${GREEN}✓ Symlink is accessible via your existing PATH${NC}"
else
    # ~/.local/bin is not in PATH, add the install dir directly for this session
    export PATH="$INSTALL_DIR:$PATH"
    echo -e "${GREEN}✓ Added $INSTALL_DIR to current session's PATH${NC}"
fi

# Show PATH warning if needed
if [ "$PATH_WARNING" = true ]; then
    echo ""
    echo -e "${YELLOW}⚠ Warning: $SYMLINK_DIR is not in your PATH${NC}"
    echo ""
    echo "To use oh-my-dot, you need to add one of the following to your PATH:"
    echo "  1. Add $SYMLINK_DIR to your PATH (recommended)"
    echo "  2. Add $INSTALL_DIR to your PATH"
    echo ""
    echo "Add one of these lines to your shell's configuration file:"
    echo "  For bash (~/.bashrc or ~/.bash_profile) or zsh (~/.zshrc):"
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
