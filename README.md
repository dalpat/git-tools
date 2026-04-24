# Git Tools

AI-powered git tools using Groq.

## Tools

- **think-commit-msg** - Generate Conventional Commits format messages from staged changes
- **think-review** - Review code changes with AI-powered feedback

## Quick Install (One-liner)

```bash
curl -sL https://raw.githubusercontent.com/dalpat/git-tools/main/install.sh | bash
```

This will:
1. Download the scripts to `~/.local/bin/`
2. Ask which tools to install
3. Ask for your Groq API key (or skip to set manually)
4. Add them to your PATH
5. Print usage instructions

## Manual Install

```bash
git clone https://github.com/dalpat/git-tools.git
cd git-tools
chmod +x think-commit-msg think-review
```

For global access:

```bash
sudo cp think-commit-msg think-review /usr/local/bin/
```

## think-commit-msg

Generate commit messages from staged changes.

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

### Options

| Flag | Description |
|------|-------------|
| `-s, --simple` | Simple commit message (not Conventional Commits) |
| `-v, --verbose` | Show staged files before output |
| `-l, --list` | List available Groq models |
| `-m, --model` | Use specific model (default: llama-3.3-70b-versatile) |
| `-h, --help` | Show help |

## think-review

Review code changes with AI-powered feedback.

```bash
# Review staged changes
git add .
think-review

# Review unstaged changes (if nothing staged)
git diff | think-review

# Review all changes with verbose output
think-review --verbose

# Focus on specific areas
think-review --security
think-review --style
think-review --best-practices

# Adjust batch size for large diffs
think-review --batch-size 5

# Retry on failure
think-review --retry 3

# Use specific model
think-review --model llama-3.3-70b-versatile
```

### Options

| Flag | Description |
|------|-------------|
| `-a, --all` | Review all focus areas (default) |
| `--security` | Focus on security vulnerabilities |
| `--style` | Focus on code style and formatting |
| `--best-practices` | Focus on best practices |
| `-b, --batch-size` | Files per batch (default: 3) |
| `-r, --retry` | Retry failed API calls (default: 2) |
| `-v, --verbose` | Show diff before review |
| `-l, --list` | List available Groq models |
| `-m, --model` | Use specific model |
| `-h, --help` | Show help |

### Configuration

Config files are read in this order (highest wins):

1. CLI flags
2. `.think-reviewrc` (project-level, create from `.think-reviewrc.example`)
3. `~/.think-tools.json` (global defaults)

Example `.think-reviewrc`:

```json
{
  "batch_size": 3,
  "retry": 2,
  "focus": {
    "security": true,
    "style": true,
    "best_practices": true
  }
}
```

Example `~/.think-tools.json`:

```json
{
  "api_key": "gsk_...",
  "model": "llama-3.3-70b-versatile"
}
```

## Configuration

### Get API Key

1. Visit https://console.groq.com
2. Create an account (free)
3. Create a new API key
4. The installer will ask for it, or set manually in `~/.think-tools.json`

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `GROQ_API_KEY` | (required) | Your Groq API key |
| `GROQ_MODEL` | `llama-3.3-70b-versatile` | Model to use |

## Requirements

- Git
- `jq`
- `curl`
- Groq API key (free)

## License

MIT