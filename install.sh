#!/bin/bash
# install.sh - Universal installer for mitl
# Usage: curl -fsSL https://mitl.run/install.sh | bash

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
BOLD='\033[1m'
RESET='\033[0m'

# Configuration
REPO="mitl-cli/mitl"
INSTALL_DIR="/usr/local/bin"
BINARY_NAME="mitl"

print_banner() {
    echo ""
    echo -e "${BOLD}🏹 mitl installer${RESET}"
    echo "  The 10x faster Docker alternative for Mac"
    echo ""
}

detect_os() {
    case "$(uname -s)" in
        Darwin*)    OS="darwin" ;;
        Linux*)     OS="linux" ;;
        *)          echo -e "${RED}Unsupported OS: $(uname -s)${RESET}"; exit 1 ;;
    esac
}

detect_arch() {
    case "$(uname -m)" in
        x86_64)     ARCH="amd64" ;;
        amd64)      ARCH="amd64" ;;
        arm64)      ARCH="arm64" ;;
        aarch64)    ARCH="arm64" ;;
        *)          echo -e "${RED}Unsupported architecture: $(uname -m)${RESET}"; exit 1 ;;
    esac
}

detect_apple_silicon() {
    if [[ "$OS" == "darwin" ]] && [[ "$ARCH" == "arm64" ]]; then
        APPLE_SILICON=true
        echo -e "${GREEN}✅ Apple Silicon detected (M1/M2/M3)${RESET}"
        echo "   You'll get maximum performance with mitl!"
    else
        APPLE_SILICON=false
    fi
}

check_requirements() {
    echo "Checking requirements..."

    if ! command -v curl &> /dev/null && ! command -v wget &> /dev/null; then
        echo -e "${RED}❌ Neither curl nor wget found${RESET}"
        echo "   Please install curl: brew install curl"
        exit 1
    fi

    if [[ "$OS" == "darwin" ]]; then
        MAC_VERSION=$(sw_vers -productVersion | cut -d. -f1)
        if [[ ${MAC_VERSION:-0} -lt 12 ]]; then
            echo -e "${YELLOW}⚠️  macOS 12+ recommended for best performance${RESET}"
        fi
    fi

    RUNTIME_FOUND=false
    if command -v container &> /dev/null; then
        echo -e "${GREEN}✅ Apple Container detected (optimal)${RESET}"
        RUNTIME_FOUND=true
    elif command -v docker &> /dev/null; then
        echo -e "${YELLOW}⚠️  Docker detected (slower than Apple Container)${RESET}"
        RUNTIME_FOUND=true
    elif command -v podman &> /dev/null; then
        echo -e "${GREEN}✅ Podman detected${RESET}"
        RUNTIME_FOUND=true
    fi

    if [[ "$RUNTIME_FOUND" == false ]]; then
        echo -e "${YELLOW}⚠️  No container runtime detected${RESET}"
        echo "   mitl requires Docker, Podman, or Apple Container"
        echo ""
        echo "   Install options:"
        if [[ "$APPLE_SILICON" == true ]]; then
            echo "   • Apple Container (recommended): developer.apple.com/virtualization"
        fi
        echo "   • Docker Desktop: brew install --cask docker"
        echo "   • Podman: brew install podman"
        echo ""
        read -p "Continue anyway? (y/n) " -n 1 -r || true
        echo
        if [[ ! ${REPLY:-n} =~ ^[Yy]$ ]]; then
            exit 1
        fi
    fi
}

get_latest_version() {
    echo "Finding latest version..."
    if command -v curl &> /dev/null; then
        VERSION=$(curl -s "https://api.github.com/repos/$REPO/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
    else
        VERSION=$(wget -qO- "https://api.github.com/repos/$REPO/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
    fi
    if [[ -z "${VERSION:-}" ]]; then
        VERSION="v0.1.0-alpha"
        echo -e "${YELLOW}Using alpha version: $VERSION${RESET}"
    else
        echo -e "Latest version: ${BOLD}$VERSION${RESET}"
    fi
}

download_binary() {
    PLATFORM="${OS}-${ARCH}"
    URL="https://github.com/$REPO/releases/download/$VERSION/mitl-$PLATFORM"

    echo "Downloading mitl for $PLATFORM..."
    TMP_DIR=$(mktemp -d)
    TMP_FILE="$TMP_DIR/mitl"

    if command -v curl &> /dev/null; then
        curl -fsSL "$URL" -o "$TMP_FILE"
    else
        wget -q "$URL" -O "$TMP_FILE"
    fi

    chmod +x "$TMP_FILE"

    if [[ ! -f "$TMP_FILE" ]]; then
        echo -e "${RED}❌ Download failed${RESET}"
        exit 1
    fi

    if ! "$TMP_FILE" version &> /dev/null; then
        echo -e "${RED}❌ Binary verification failed${RESET}"
        exit 1
    fi

    echo -e "${GREEN}✅ Download successful${RESET}"
}

install_binary() {
    echo "Installing mitl to $INSTALL_DIR..."
    if [[ -w "$INSTALL_DIR" ]]; then
        mv "$TMP_FILE" "$INSTALL_DIR/$BINARY_NAME"
    else
        echo "Permission required to install to $INSTALL_DIR"
        sudo mv "$TMP_FILE" "$INSTALL_DIR/$BINARY_NAME"
    fi

    if command -v mitl &> /dev/null; then
        echo -e "${GREEN}✅ Installation successful!${RESET}"
        INSTALLED_VERSION=$(mitl version 2>/dev/null | head -1)
        echo -e "   Installed: ${BOLD}$INSTALLED_VERSION${RESET}"
    else
        echo -e "${RED}❌ Installation failed${RESET}"
        exit 1
    fi

    rm -rf "$TMP_DIR"
}

setup_completion() {
    echo "Setting up shell completion..."

    if [[ -n "${BASH_VERSION:-}" ]]; then
        COMPLETION_FILE="$HOME/.bash_completion.d/mitl"
        mkdir -p "$(dirname "$COMPLETION_FILE")"
        mitl completion bash > "$COMPLETION_FILE"
        if ! grep -q "\.bash_completion.d/mitl" "$HOME/.bashrc" 2>/dev/null; then
            echo "source $COMPLETION_FILE" >> "$HOME/.bashrc"
        fi
        echo -e "${GREEN}✅ Shell completion installed for bash${RESET}"
    elif [[ -n "${ZSH_VERSION:-}" ]]; then
        COMPLETION_FILE="$HOME/.zsh/completions/_mitl"
        mkdir -p "$(dirname "$COMPLETION_FILE")"
        mitl completion zsh > "$COMPLETION_FILE"
        if ! grep -q "fpath.*\.zsh/completions" "$HOME/.zshrc" 2>/dev/null; then
            echo 'fpath=($HOME/.zsh/completions $fpath)' >> "$HOME/.zshrc"
            echo 'autoload -Uz compinit && compinit' >> "$HOME/.zshrc"
        fi
        echo -e "${GREEN}✅ Shell completion installed for zsh${RESET}"
    fi
}

post_install_message() {
    echo ""
    echo -e "${BOLD}${GREEN}🎉 mitl installed successfully!${RESET}"
    echo ""
    echo -e "${BOLD}Next steps:${RESET}"
    echo "1. Check your setup:"
    echo -e "   ${BLUE}mitl doctor${RESET}"
    echo ""
    echo "2. Try it out:"
    echo -e "   ${BLUE}mitl run echo 'Hello from mitl!'${RESET}"
    echo ""
    if [[ "$APPLE_SILICON" == true ]] && ! command -v container &> /dev/null; then
        echo -e "${BOLD}${YELLOW}Performance tip for Apple Silicon:${RESET}"
        echo "Install Apple Container for 5-10x speed boost:"
        echo "  https://developer.apple.com/virtualization"
        echo ""
    fi
    echo -e "${BOLD}Documentation:${RESET} https://github.com/mitl-cli/mitl"
    echo -e "${BOLD}Report issues:${RESET} https://github.com/mitl-cli/mitl/issues"
    echo ""
}

main() {
    print_banner
    detect_os
    detect_arch
    detect_apple_silicon
    check_requirements
    get_latest_version
    download_binary
    install_binary
    setup_completion
    post_install_message
}

main "$@"

