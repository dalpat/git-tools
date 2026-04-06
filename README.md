# Git Commit Message Generator

Generates smart commit messages using AI (Groq).

## Installation

```bash
git clone https://github.com/dalpat/git-tools.git
cd git-tools
chmod +x think-commit-msg
```

## Setup

Add to your `~/.bashrc` or `~/.zshrc`:

```bash
export GROQ_API_KEY="your-key"
```

Then run: `source ~/.bashrc` or `source ~/.zshrc`

Get your free API key at: https://console.groq.com

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
- Groq API key (free at https://console.groq.com)

## Configuration

Optional environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `GROQ_MODEL` | `llama-3.3-70b-versatile` | Groq model to use |

## Features

- Conventional Commits format by default
- Clean output for safe scripting
- Validates API key before making requests
- Handles API errors gracefully