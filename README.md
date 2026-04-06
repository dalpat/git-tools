# Git Commit Message Generator

Generates smart commit messages using AI (Gemini or Groq).

## Usage

```bash
# Stage your changes first
git add .

# Run the script
./think-commit-msg
```

## Configuration

Set environment variables:

```bash
# Required (choose one)
export GEMINI_API_KEY="your-key"   # Gemini (default)
export GROQ_API_KEY="your-key"     # Groq

# Optional
export MODEL_API="gemini"          # or "groq" (default: gemini)
export GROQ_MODEL="mixtral-8x7b-32768"  # Groq model
```

## Requirements

- Git with staged changes
- `jq`
- `curl`
- API key (Gemini or Groq)

## Features

- Supports both Gemini and Groq providers
- Validates API keys before making requests
- Handles API errors gracefully