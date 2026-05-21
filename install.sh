#!/bin/bash

set -euo pipefail

# Check required dependencies
if ! command -v jq &> /dev/null; then
    echo "Error: jq is required but not installed."
    echo "Install it first:"
    echo "  Ubuntu/Debian:  sudo apt-get install jq"
    echo "  macOS:          brew install jq"
    echo "  Fedora:         sudo dnf install jq"
    echo "  Arch:           sudo pacman -S jq"
    exit 1
fi

REPO_URL="https://raw.githubusercontent.com/dalpat/git-tools/main"
INSTALL_DIR="${HOME}/.local/bin"
# Detect the correct shell config file
SHELL_NAME=$(basename "${SHELL:-bash}")
case "$SHELL_NAME" in
    zsh)
        CONFIG_FILE="${HOME}/.zshrc"
        ;;
    bash)
        CONFIG_FILE="${HOME}/.bashrc"
        if [[ ! -f "$CONFIG_FILE" ]] && [[ -f "${HOME}/.bash_profile" ]]; then
            CONFIG_FILE="${HOME}/.bash_profile"
        fi
        ;;
    fish)
        CONFIG_FILE="${HOME}/.config/fish/config.fish"
        ;;
    *)
        CONFIG_FILE="${HOME}/.bashrc"
        ;;
esac
SHARED_CONFIG="${HOME}/.think-tools.json"
EXAMPLE_CONFIG="${PWD}/.think-reviewrc.example"
CACHE_DIR=".cache/think-review"

echo "============================================"
echo "  Git Tools Installer"
echo "============================================"
echo ""

echo "This installer can set up the following tools:"
echo "  1. think-commit-msg - Generate commit messages"
echo "  2. think-review     - Review code changes"
echo "  3. think-git-graph  - Visual git history viewer"
echo "  4. All tools"
echo ""

read -p "Which tool do you want to install? [1/2/3/4/all]: " CHOICE

# Normalize input: strip whitespace and lowercase
CHOICE=$(echo "$CHOICE" | tr -d '[:space:]' | tr '[:upper:]' '[:lower:]')

INSTALL_COMMIT=false
INSTALL_REVIEW=false
INSTALL_GRAPH=false

case "$CHOICE" in
    1)
        echo "OK, will install think-commit-msg"
        INSTALL_COMMIT=true
        ;;
    2)
        echo "OK, will install think-review"
        INSTALL_REVIEW=true
        ;;
    3)
        echo "OK, will install think-git-graph"
        INSTALL_GRAPH=true
        ;;
    4|all|a)
        echo "OK, will install all tools"
        INSTALL_COMMIT=true
        INSTALL_REVIEW=true
        INSTALL_GRAPH=true
        ;;
    *)
        echo "Invalid choice. Installing all tools by default."
        INSTALL_COMMIT=true
        INSTALL_REVIEW=true
        INSTALL_GRAPH=true
        ;;
esac

echo ""

if [[ -f "$SHARED_CONFIG" ]]; then
    echo "Found existing $SHARED_CONFIG"
    EXISTING_KEY=$(jq -r '.api_key // empty' "$SHARED_CONFIG" 2>/dev/null || echo "")
    EXISTING_MODEL=$(jq -r '.model // empty' "$SHARED_CONFIG" 2>/dev/null || echo "")
else
    EXISTING_KEY=""
    EXISTING_MODEL=""
fi

if [[ -n "$EXISTING_KEY" ]]; then
    echo "Using existing API key from config"
    USE_EXISTING=true
else
    USE_EXISTING=false
fi

API_KEY=""
if [[ "$USE_EXISTING" == "false" ]]; then
    read -p "Enter your Groq API key (or press Enter to skip): " API_KEY
    
    if [[ -z "$API_KEY" ]]; then
        echo "No API key provided. You can set it later in $SHARED_CONFIG:"
        echo "  {\"api_key\": \"your-key\", \"model\": \"llama-3.3-70b-versatile\"}"
        echo ""
    else
        echo "{
  \"api_key\": \"$API_KEY\",
  \"model\": \"llama-3.3-70b-versatile\"
}" > "$SHARED_CONFIG"
        echo "Created $SHARED_CONFIG with API key"
    fi
fi

DEFAULT_MODEL="llama-3.3-70b-versatile"
read -p "Enter Groq model (default: $DEFAULT_MODEL): " MODEL
MODEL="${MODEL:-$DEFAULT_MODEL}"

if [[ ! -f "$SHARED_CONFIG" ]] || [[ "$USE_EXISTING" == "false" ]]; then
    if [[ -n "$EXISTING_KEY" ]]; then
        jq -s '.[0] * .[1]' \
            <(jq -n "{\"api_key\": \"$EXISTING_KEY\"}") \
            <(jq -n "{\"model\": \"$MODEL\"}") > "${SHARED_CONFIG}.tmp" 2>/dev/null || \
        echo "{
  \"api_key\": \"$EXISTING_KEY\",
  \"model\": \"$MODEL\"
}" > "$SHARED_CONFIG"
        mv "${SHARED_CONFIG}.tmp" "$SHARED_CONFIG" 2>/dev/null || true
    elif [[ -z "$API_KEY" ]]; then
        true
    else
        echo "{
  \"api_key\": \"$API_KEY\",
  \"model\": \"$MODEL\"
}" > "$SHARED_CONFIG"
    fi
else
    TEMP_KEY=$(jq -r '.api_key // empty' "$SHARED_CONFIG" 2>/dev/null || echo "")
    TEMP_MODEL=$(jq -r '.model // empty' "$SHARED_CONFIG" 2>/dev/null || echo "$MODEL")
    if [[ -n "$TEMP_KEY" ]]; then
        echo "{
  \"api_key\": \"$TEMP_KEY\",
  \"model\": \"$TEMP_MODEL\"
}" > "$SHARED_CONFIG"
    fi
fi

echo ""

mkdir -p "$INSTALL_DIR"

if [[ "$INSTALL_COMMIT" == "true" ]]; then
    echo "Installing think-commit-msg..."
    curl -fsSL "${REPO_URL}/think-commit-msg" -o "${INSTALL_DIR}/think-commit-msg"
    chmod +x "${INSTALL_DIR}/think-commit-msg"
    echo "think-commit-msg installed"
fi

if [[ "$INSTALL_REVIEW" == "true" ]]; then
    echo "Installing think-review..."
    curl -fsSL "${REPO_URL}/think-review" -o "${INSTALL_DIR}/think-review"
    chmod +x "${INSTALL_DIR}/think-review"

    echo "Installing think-tools-lib.sh..."
    curl -fsSL "${REPO_URL}/think-tools-lib.sh" -o "${INSTALL_DIR}/think-tools-lib.sh"
    chmod +x "${INSTALL_DIR}/think-tools-lib.sh"
    echo "think-review and think-tools-lib.sh installed"
fi

if [[ "$INSTALL_GRAPH" == "true" ]]; then
    echo "Installing think-git-graph..."

    if command -v go &> /dev/null; then
        SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

        if [[ -f "${SCRIPT_DIR}/think-git-graph/go.mod" ]]; then
            echo "Building from local source..."
            (cd "${SCRIPT_DIR}/think-git-graph" && go build -o "${INSTALL_DIR}/think-git-graph" .)
        else
            echo "Downloading source and building..."
            TMP_DIR=$(mktemp -d)
            curl -fsL "https://github.com/dalpat/git-tools/archive/refs/heads/main.tar.gz" -o "${TMP_DIR}/git-tools.tar.gz"
            tar -xzf "${TMP_DIR}/git-tools.tar.gz" -C "${TMP_DIR}"
            (cd "${TMP_DIR}/git-tools-main/think-git-graph" && go build -o "${INSTALL_DIR}/think-git-graph" .)
            rm -rf "${TMP_DIR}"
        fi
        chmod +x "${INSTALL_DIR}/think-git-graph"
        echo "think-git-graph built and installed"
    else
        echo "Go not found. Downloading pre-built binary..."
        GO_BIN_URL="${REPO_URL}/think-git-graph/think-git-graph"
        # -f ensures curl fails on 404, so we don't create an empty executable
        if curl -fsSL "${GO_BIN_URL}" -o "${INSTALL_DIR}/think-git-graph"; then
            chmod +x "${INSTALL_DIR}/think-git-graph"
            echo "think-git-graph installed"
        else
            echo "Error: No pre-built binary available and Go is not installed."
            echo "Install Go (https://go.dev/dl/) and run this installer again."
            exit 1
        fi
    fi
fi

if [[ ":$PATH:" != *":${INSTALL_DIR}:"* ]]; then
    # Ensure parent directory exists (e.g. for fish config)
    mkdir -p "$(dirname "$CONFIG_FILE")"
    if ! grep -q "export PATH=.*\.local/bin" "$CONFIG_FILE" 2>/dev/null; then
        echo "" >> "$CONFIG_FILE"
        echo "export PATH=\"\$PATH:\${HOME}/.local/bin\"" >> "$CONFIG_FILE"
        echo "Added to PATH in $CONFIG_FILE"
    fi
    export PATH="${PATH}:${INSTALL_DIR}"
fi

echo ""

if [[ "$INSTALL_REVIEW" == "true" ]]; then
    echo "{
  \"batch_size\": 3,
  \"retry\": 2,
  \"focus\": {
    \"security\": true,
    \"style\": true,
    \"best_practices\": true
  }
}" > "$EXAMPLE_CONFIG"
    echo "Created $EXAMPLE_CONFIG template"
    
    if [[ ! -d "$CACHE_DIR" ]]; then
        mkdir -p "$CACHE_DIR"
        echo "Created $CACHE_DIR directory"
    fi
    
    if [[ -f ".gitignore" ]]; then
        if ! grep -q "$CACHE_DIR" ".gitignore" 2>/dev/null; then
            echo "$CACHE_DIR/" >> ".gitignore"
            echo "Added $CACHE_DIR to .gitignore"
        fi
    elif [[ -d ".git" ]]; then
        echo "$CACHE_DIR/" >> ".gitignore"
        echo "Added $CACHE_DIR to .gitignore"
    fi
fi

echo ""
echo "============================================"
echo "  Installation complete!"
echo "============================================"
echo ""

echo "Usage:"
echo "  1. Restart your terminal or run: source $CONFIG_FILE"

if [[ "$INSTALL_COMMIT" == "true" ]]; then
    echo "  2. Generate commit: think-commit-msg"
    echo "     git add . && git commit -m \"\$(think-commit-msg)\""
fi

if [[ "$INSTALL_REVIEW" == "true" ]]; then
    echo "  2. Review changes: think-review"
    echo "     think-review --help"
fi

if [[ "$INSTALL_GRAPH" == "true" ]]; then
    echo "  2. Visual git graph: think-git-graph"
    echo "     think-git-graph --detach"
fi

echo ""
