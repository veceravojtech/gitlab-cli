#!/bin/bash
# Installation script for gitlab-cli
# Builds and installs the CLI to ~/.local/bin

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

INSTALL_DIR="${HOME}/.local/bin"
BINARY_NAME="gitlab-cli"
VERSION="1.0.0"
BUILD_TIME=$(date -u '+%Y-%m-%d_%H:%M:%S')

echo -e "${GREEN}=== GitLab CLI Installation ===${NC}"

# Check if Go is installed
if ! command -v go &> /dev/null; then
    echo -e "${RED}Error: Go is not installed.${NC}"
    echo "Please install Go from https://go.dev/dl/"
    exit 1
fi

GO_VERSION=$(go version | awk '{print $3}')
echo -e "Found Go: ${GREEN}${GO_VERSION}${NC}"

# Get the script directory (project root)
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

echo "Building from: $SCRIPT_DIR"

# Download dependencies
echo -e "${YELLOW}Downloading dependencies...${NC}"
go mod download

# Build the binary with version info
echo -e "${YELLOW}Building ${BINARY_NAME} v${VERSION}...${NC}"
LDFLAGS="-X github.com/user/gitlab-cli/internal/cli.Version=${VERSION} -X github.com/user/gitlab-cli/internal/cli.BuildTime=${BUILD_TIME}"
go build -ldflags "${LDFLAGS}" -o "${BINARY_NAME}" ./cmd/gitlab-cli

if [ ! -f "${BINARY_NAME}" ]; then
    echo -e "${RED}Error: Build failed - binary not created${NC}"
    exit 1
fi

# Create install directory if it doesn't exist
if [ ! -d "$INSTALL_DIR" ]; then
    echo -e "${YELLOW}Creating ${INSTALL_DIR}...${NC}"
    mkdir -p "$INSTALL_DIR"
fi

# Install the binary
echo -e "${YELLOW}Installing to ${INSTALL_DIR}/${BINARY_NAME}...${NC}"
mv "${BINARY_NAME}" "${INSTALL_DIR}/"
chmod +x "${INSTALL_DIR}/${BINARY_NAME}"

echo -e "${GREEN}Installation complete!${NC}"

# Check if INSTALL_DIR is in PATH
if [[ ":$PATH:" != *":$INSTALL_DIR:"* ]]; then
    echo ""
    echo -e "${YELLOW}Warning: ${INSTALL_DIR} is not in your PATH${NC}"
    echo "Add this line to your ~/.bashrc or ~/.zshrc:"
    echo ""
    echo -e "  ${GREEN}export PATH=\"\$HOME/.local/bin:\$PATH\"${NC}"
    echo ""
    echo "Then run: source ~/.bashrc  (or source ~/.zshrc)"
fi

# Check for required environment variables
echo ""
echo -e "${YELLOW}=== Configuration ===${NC}"
echo "Set these environment variables in your shell profile:"
echo ""

if [ -z "$GITLAB_URL" ]; then
    echo -e "  ${RED}GITLAB_URL${NC} (not set)"
    echo "    export GITLAB_URL=\"https://gitlab.example.com\""
else
    echo -e "  ${GREEN}GITLAB_URL${NC} = $GITLAB_URL"
fi

if [ -z "$GITLAB_TOKEN" ]; then
    echo -e "  ${RED}GITLAB_TOKEN${NC} (not set)"
    echo "    export GITLAB_TOKEN=\"your-private-token\""
else
    echo -e "  ${GREEN}GITLAB_TOKEN${NC} = [set]"
fi

echo ""
echo -e "${GREEN}Run '${BINARY_NAME} version' to verify installation${NC}"
echo -e "${GREEN}Run '${BINARY_NAME} --help' to get started${NC}"
