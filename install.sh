#!/bin/bash

set -e

REPO_URL="https://raw.githubusercontent.com/dalpat/git-tools/main"
INSTALL_DIR="${HOME}/.local/bin"
CONFIG_FILE="${HOME}/.bashrc"

echo "============================================"
echo "  Git Commit Message Generator Installer"
echo "============================================"
echo ""

echo "This will install think-commit-msg tool."
echo ""

read -p "Enter your Groq API key (or press Enter to skip): " API_KEY

if [[ -z "$API_KEY" ]]; then
    echo "No API key provided. You can set it later by adding to your ~/.bashrc:"
    echo "  export GROQ_API_KEY='your-key'"
    echo ""
else
    if ! grep -q "export GROQ_API_KEY=" "$CONFIG_FILE" 2>/dev/null; then
        echo "" >> "$CONFIG_FILE"
        echo "# Groq API key for think-commit-msg" >> "$CONFIG_FILE"
        echo "export GROQ_API_KEY='$API_KEY'" >> "$CONFIG_FILE"
        echo "Added GROQ_API_KEY to $CONFIG_FILE"
    else
        echo "GROQ_API_KEY already exists in $CONFIG_FILE, skipping..."
    fi
fi

echo ""
echo "Installing think-commit-msg..."

mkdir -p "$INSTALL_DIR"

curl -sL "${REPO_URL}/think-commit-msg" -o "${INSTALL_DIR}/think-commit-msg"
chmod +x "${INSTALL_DIR}/think-commit-msg"

if [[ ":$PATH:" == *":${INSTALL_DIR}:"* ]]; then
    echo "PATH already contains $INSTALL_DIR"
else
    if ! grep -q "export PATH=.*\.local/bin" "$CONFIG_FILE" 2>/dev/null; then
        echo "" >> "$CONFIG_FILE"
        echo "export PATH=\"\$PATH:\${HOME}/.local/bin\"" >> "$CONFIG_FILE"
        echo "Added to PATH in $CONFIG_FILE"
    fi
    export PATH="${PATH}:${HOME}/.local/bin"
fi

echo ""
echo "============================================"
echo "  Installation complete!"
echo "============================================"
echo ""
echo "Usage:"
echo "  1. Restart your terminal or run: source ~/.bashrc"
echo "  2. Stage your changes: git add ."
echo "  3. Generate commit: think-commit-msg"
echo ""
echo "Quick commit:"
echo "  git add . && git commit -m \"\$(think-commit-msg)\""
echo ""
echo "More options:"
echo "  think-commit-msg --help"
echo "  think-commit-msg --list"
echo ""