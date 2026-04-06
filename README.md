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
# For Gemini (default)
export GEMINI_API_KEY="your-key"

# Or for Groq
export GROQ_API_KEY="your-key"
export MODEL_API="groq"
```

Then run: `source ~/.bashrc` or `source ~/.zshrc`

## Usage

```bash
# Stage your changes
git add .

# Run the script
think-commit-msg

# Use Conventional Commits format
think-commit-msg --conventional
```

The script outputs the suggested commit message - copy it or run:
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
| `MODEL_API` | `gemini` | Provider: `gemini` or `groq` |
| `GROQ_MODEL` | `mixtral-8x7b-32768` | Groq model to use |

## Features

- Supports both Gemini and Groq providers
- Validates API keys before making requests
- Handles API errors gracefully