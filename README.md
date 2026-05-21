# Git Tools

AI-powered git tools using Groq.

## Tools

- **think-commit-msg** - Generate Conventional Commits format messages from staged changes
- **think-review** - Review code changes with AI-powered feedback
- **think-git-graph** - Visual git history browser GUI with colored branch graph

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
# Or build think-git-graph from source:
cd think-git-graph && go build -o ~/.local/bin/think-git-graph .
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

## think-git-graph

Visual git history browser GUI. Shows colored branch graph, commit details, and current branch.

```bash
# Open git graph in browser (blocks terminal)
think-git-graph

# Run in background
think-git-graph --detach

# Stop detached instance
kill $(cat /tmp/think-git-graph.pid)
```

### Options

| Flag | Description |
|------|-------------|
| `--detach` | Run server in background with PID file |
| `--port` | Port to listen on (default: random) |
| `--host` | Host to bind to (default: 127.0.0.1) |
| `--no-open` | Do not open browser automatically |
| `-h, --help` | Show help |

### Features

- **Git log --graph style layout** — dynamic lane assignment with curved bezier connections between lanes
- **Colored branch graph** — each branch gets a distinct color, rendered on HTML Canvas
- **Merge visualization** — double-ring commit nodes for merge commits with cross-lane bezier curves
- **Click any commit** — slide-in detail panel showing full message, author, date, and files changed with +/- stats
- **Current branch indicator** — green dot (clean) or yellow dot (uncommitted changes)
- **Pagination** — loads 200 commits by default, "Load More" button fetches older commits
- **Refresh button** — manually update the graph after making changes in another terminal
- **Dark & light themes** — clean, minimal UI with amber accents; respects system preference
- **Auto-refresh** — graph updates automatically when git changes are detected
- **Zero runtime dependencies** — single compiled Go binary, no npm, no node_modules
- **`--detach` mode** — runs in background with PID file, parent waits for URL then exits

### How to Use

```bash
# Quick start — opens in your default browser
cd /path/to/any/git/repo
think-git-graph

# Background mode — returns immediately, browser opens automatically
think-git-graph --detach

# Run on a fixed port (so you can bookmark it)
think-git-graph --port 8080

# Run without auto-opening the browser (useful in SSH/headless/WSL)
think-git-graph --no-open

# Combine flags for background + fixed port
think-git-graph --detach --port 8080 --host 0.0.0.0

# View the listening URL
cat /tmp/think-git-graph.url

# Stop the background server
kill $(cat /tmp/think-git-graph.pid)

# Build from source (requires Go 1.22+)
cd think-git-graph
go build -o ~/.local/bin/think-git-graph .
```

### Building from Source

```bash
git clone https://github.com/dalpat/git-tools.git
cd git-tools/think-git-graph
go build -o ~/.local/bin/think-git-graph .
```

### Requirements

- Git
- Go 1.22+ (only if building from source)
- A web browser (Chrome, Firefox, Safari, Edge)

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

| Tool | Dependencies |
|------|-------------|
| think-commit-msg | Git, jq, curl, Groq API key |
| think-review | Git, jq, curl, Groq API key |
| think-git-graph | Git, Go 1.22+ (build only), web browser |

## License

MIT