# Git Commit Message Generator

Generates smart commit messages using AI (Gemini or Groq).

## Installation

```bash
git clone https://github.com/dalpat/git-tools.git
cd git-tools
chmod +x think-commit-msg
```

## Setup

Add to your `~/.bashrc` or `~/.zshrc` (choose one):

```bash
# For Groq (default)
export GROQ_API_KEY="your-key"

# Or for Gemini
export GEMINI_API_KEY="your-key"
export MODEL_API="gemini"
```

Then run: `source ~/.bashrc` or `source ~/.zshrc`

## Usage

```bash
# Stage your changes
git add .

# Run the script (Conventional Commits format by default)
think-commit-msg

# Use simple format instead
think-commit-msg --simple

# Show staged files before commit message
think-commit-msg --verbose
```

The script outputs only the commit message (safe for scripting):
```bash
git add . && git commit -m "$(think-commit-msg)"
```

## Global Installation (optional)

```bash
sudo cp think-commit-msg /usr/local/bin/think-commit-msg

# Then run anywhere
think-commit-msg
```

## Requirements

- Git with staged changes
- `jq`
- `curl`
- API key (Gemini or Groq)

## Configuration

Optional environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `MODEL_API` | `groq` | Provider: `gemini` or `groq` |
| `GROQ_MODEL` | `llama-3.3-70b-versatile` | Groq model to use |

## Features

- Supports both Gemini and Groq providers
- Conventional Commits format by default
- Validates API keys before making requests
- Handles API errors gracefully
- Clean output for safe scripting