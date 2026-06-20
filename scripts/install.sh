#!/bin/bash

# AICockpit Installation Script
# This script handles user-level installation and PATH configuration

set -e

BINARY_NAME="cockpit"
BINARY_PATH="bin/${BINARY_NAME}"
INSTALL_PATH="${HOME}/.local/bin"
COCKPIT_PATH="${INSTALL_PATH}/${BINARY_NAME}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}=== AICockpit Installation ===${NC}\n"

# Check if binary exists
if [ ! -f "$BINARY_PATH" ]; then
    echo -e "${RED}Error: Binary not found at $BINARY_PATH${NC}"
    echo "Please run 'make build' first"
    exit 1
fi

# Create install directory
echo -e "${BLUE}Creating installation directory...${NC}"
mkdir -p "$INSTALL_PATH"

# Copy binary
echo -e "${BLUE}Installing binary...${NC}"
cp "$BINARY_PATH" "$COCKPIT_PATH"
chmod +x "$COCKPIT_PATH"
echo -e "${GREEN}✓ Binary installed to $COCKPIT_PATH${NC}\n"

# Function to check if PATH already contains the directory
path_already_added() {
    local shell_config="$1"
    if [ -f "$shell_config" ]; then
        grep -q "\.local/bin" "$shell_config" 2>/dev/null && return 0
    fi
    return 1
}

# Function to add to PATH in shell config
add_to_path() {
    local shell_config="$1"
    local shell_name="$2"
    
    if [ ! -f "$shell_config" ]; then
        echo -e "${YELLOW}Creating $shell_config${NC}"
        touch "$shell_config"
    fi
    
    if path_already_added "$shell_config"; then
        echo -e "${YELLOW}✓ $shell_name: PATH already configured${NC}"
        return 0
    fi
    
    echo -e "${BLUE}Adding to $shell_name...${NC}"
    echo "" >> "$shell_config"
    echo "# AICockpit - Added by installation script" >> "$shell_config"
    echo "export PATH=\"\$HOME/.local/bin:\$PATH\"" >> "$shell_config"
    echo -e "${GREEN}✓ Added to $shell_config${NC}"
}

# Detect and configure shells
echo -e "${BLUE}Configuring shell...${NC}\n"

# Bash
if [ -n "$BASH_VERSION" ] || [ -f "$HOME/.bashrc" ]; then
    add_to_path "$HOME/.bashrc" "Bash"
fi

# Zsh
if [ -n "$ZSH_VERSION" ] || [ -f "$HOME/.zshrc" ]; then
    add_to_path "$HOME/.zshrc" "Zsh"
fi

# Fish
if [ -f "$HOME/.config/fish/config.fish" ]; then
    if ! grep -q "\.local/bin" "$HOME/.config/fish/config.fish" 2>/dev/null; then
        echo -e "${BLUE}Adding to Fish...${NC}"
        echo "" >> "$HOME/.config/fish/config.fish"
        echo "# AICockpit - Added by installation script" >> "$HOME/.config/fish/config.fish"
        echo "set -gx PATH \$HOME/.local/bin \$PATH" >> "$HOME/.config/fish/config.fish"
        echo -e "${GREEN}✓ Added to Fish${NC}"
    else
        echo -e "${YELLOW}✓ Fish: PATH already configured${NC}"
    fi
fi

echo ""

# Check if PATH is already set in current session
if echo "$PATH" | grep -q ".local/bin"; then
    echo -e "${GREEN}✓ ~/.local/bin is already in your PATH${NC}"
    RELOAD_NEEDED=false
else
    echo -e "${YELLOW}⚠ ~/.local/bin is not in your current PATH${NC}"
    echo -e "${YELLOW}  You need to reload your shell configuration${NC}"
    RELOAD_NEEDED=true
fi

echo ""
echo -e "${GREEN}=== Installation Complete ===${NC}\n"

# Verify installation
if command -v cockpit &> /dev/null; then
    VERSION=$($COCKPIT_PATH --version)
    echo -e "${GREEN}✓ $VERSION${NC}"
    echo -e "${GREEN}✓ cockpit is ready to use!${NC}"
else
    if [ "$RELOAD_NEEDED" = true ]; then
        echo -e "${YELLOW}To use cockpit, reload your shell:${NC}"
        echo ""
        if [ -n "$BASH_VERSION" ]; then
            echo "  source ~/.bashrc"
        elif [ -n "$ZSH_VERSION" ]; then
            echo "  source ~/.zshrc"
        else
            echo "  source ~/.bashrc  # or ~/.zshrc or ~/.config/fish/config.fish"
        fi
        echo ""
        echo -e "${YELLOW}Or open a new terminal window${NC}"
    else
        echo -e "${YELLOW}cockpit not found in PATH${NC}"
        echo -e "${YELLOW}Try: export PATH=\"\$HOME/.local/bin:\$PATH\"${NC}"
    fi
fi

echo ""
echo -e "${BLUE}Next steps:${NC}"
echo "  1. cockpit setup    # Run the setup wizard"
echo "  2. cockpit doctor   # Verify installation"
echo "  3. cockpit info     # View configuration"
echo ""
