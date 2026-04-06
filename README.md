# Git Commit Message Generator

AI-powered git commit message generator using Groq. Generates Conventional Commits format messages from your staged changes.

## Quick Start

```bash
# 1. Clone
git clone https://github.com/dalpat/git-tools.git
cd git-tools

# 2. Get free API key
# Visit https://console.groq.com and create a key

# 3. Set API key (add to ~/.bashrc or ~/.zshrc to persist)
export GROQ_API_KEY="your-key-here"

# 4. Make executable
chmod +x think-commit-msg

# 5. Use it
git add .
think-commit-msg
```

## Installation

### Local (in project)

```bash
git clone https://github.com/dalpat/git-tools.git
cd git-tools
chmod +x think-commit-msg
```

### Global (anywhere)

```bash
sudo cp think-commit-msg /usr/local/bin/
think-commit-msg  # Run from anywhere
```

## Usage

```bash
# Generate commit message (Conventional Commits format)
git add .
think-commit-msg

# Quick commit in one line
git add . && git commit -m "$(think-commit-msg)"

# Simple format (not Conventional Commits)
think-commit-msg --simple

# Show staged files before message
think-commit-msg --verbose

# List available models
think-commit-msg --list

# Use specific model
think-commit-msg --model llama-3.1-8b-instant
```

## Options

| Flag | Description |
|------|-------------|
| `-s, --simple` | Simple commit message (not Conventional Commits) |
| `-v, --verbose` | Show staged files before output |
| `-l, --list` | List available Groq models |
| `-m, --model` | Use specific model (default: llama-3.3-70b-versatile) |
| `-h, --help` | Show help |

## Configuration

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `GROQ_API_KEY` | (required) | Your Groq API key |
| `GROQ_MODEL` | `llama-3.3-70b-versatile` | Model to use |

### Get API Key

1. Visit https://console.groq.com
2. Create an account (free)
3. Create a new API key
4. Export it: `export GROQ_API_KEY="your-key"`

## Requirements

- Git
- `jq`
- `curl`
- Groq API key (free)

## Examples

```
$ git add .
$ think-commit-msg
feat(auth): add login endpoint

$ git add .
$ think-commit-msg --simple
Add login endpoint

$ git add . && git commit -m "$(think-commit-msg)"
[main abc123] feat(auth): add login endpoint
```

## License

MIT