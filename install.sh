#!/bin/bash

set -e

REPO_URL="https://raw.githubusercontent.com/dalpat/git-tools/main"
INSTALL_DIR="${HOME}/.local/bin"
CONFIG_FILE="${HOME}/.bashrc"
SHARED_CONFIG="${HOME}/.think-tools.json"
EXAMPLE_CONFIG="${PWD}/.think-reviewrc.example"
CACHE_DIR=".cache/think-review"

echo "============================================"
echo "  Git Tools Installer"
echo "============================================"
echo ""

echo "This installer can set up the following tools:"
echo "  1. think-commit-msg - Generate commit messages"
echo "  2. think-review   - Review code changes"
echo "  3. Both tools"
echo ""

read -p "Which tool do you want to install? [1/2/3/both]: " CHOICE

case "$CHOICE" in
    1|both)
        echo "OK, will install think-commit-msg"
        ;;
    2)
        echo "OK, will install think-review"
        ;;
    3)
        echo "OK, will install both tools"
        ;;
    *)
        echo "Invalid choice. Installing both tools by default."
        ;;
esac

INSTALL_COMMIT=false
INSTALL_REVIEW=false

if [[ "$CHOICE" == "1" ]] || [[ "$CHOICE" == "b" ]]; then
    INSTALL_COMMIT=true
fi
if [[ "$CHOICE" == "2" ]] || [[ "$CHOICE" == "b" ]]; then
    INSTALL_REVIEW=true
fi
if [[ -z "$CHOICE" ]] || [[ "$CHOICE" == "3" ]]; then
    INSTALL_COMMIT=true
    INSTALL_REVIEW=true
fi

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

read -p "Enter Groq model (default: llama-3.3-70b-versatile): " MODEL
MODEL="${MODEL:-llama-3.3-70b-versatile}"

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
    curl -sL "${REPO_URL}/think-commit-msg" -o "${INSTALL_DIR}/think-commit-msg"
    chmod +x "${INSTALL_DIR}/think-commit-msg"
    echo "think-commit-msg installed"
fi

if [[ "$INSTALL_REVIEW" == "true" ]]; then
    echo "Installing think-review..."
    curl -sL "${REPO_URL}/think-review" -o "${INSTALL_DIR}/think-review"
    chmod +x "${INSTALL_DIR}/think-review"
    
    echo "Installing think-tools-lib.sh..."
    curl -sL "${REPO_URL}/think-tools-lib.sh" -o "${INSTALL_DIR}/think-tools-lib.sh"
    chmod +x "${INSTALL_DIR}/think-tools-lib.sh"
    echo "think-review and think-tools-lib.sh installed"
fi

if [[ ":$PATH:" != *":${INSTALL_DIR}:"* ]]; then
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
echo "  1. Restart your terminal or run: source ~/.bashrc"

if [[ "$INSTALL_COMMIT" == "true" ]]; then
    echo "  2. Generate commit: think-commit-msg"
    echo "     git add . && git commit -m \"\$(think-commit-msg)\""
fi

if [[ "$INSTALL_REVIEW" == "true" ]]; then
    echo "  2. Review changes: think-review"
    echo "     think-review --help"
fi

echo ""