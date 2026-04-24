- #2: Extract shared think-tools-lib.sh library - COMPLETED
  - Created think-tools-lib.sh with load_config, call_groq, stream_response, retry_on_failure functions
  - Refactored think-commit-msg to source the library with fallback for backwards compatibility
  - Verified --help works, syntax checks pass

- #3: think-review: Core tool - diff reading, API call, markdown output - COMPLETED
  - Created think-review script with core functionality
  - Reads git diffs (staged changes, falls back to unstaged if none staged)
  - Supports --help, --list, --model flags
  - Outputs markdown format with file:line references
  - Verified --help works, syntax checks pass

- #4: think-review: Config system - ~/.think-tools.json + .think-reviewrc + CLI overrides - COMPLETED
  - Added load_project_config() to think-tools-lib.sh
  - Added ensure_review_cache() to create .cache/think-review/ directory
  - think-review now reads .think-reviewrc for batch_size, retry, focus settings
  - Config resolution: CLI flags > .think-reviewrc > ~/.think-tools.json
  - Added --retry and --batch-size CLI flags
  - Created .cache/think-review/ directory on first run
  - Verified --help shows config resolution order

- #5: think-review: Focus area flags + batch processing - COMPLETED
  - Added --all, --security, --style, --best-practices CLI flags
  - Multiple focus flags can be combined (e.g., --security --style)
  - Added per-file batched review with configurable --batch-size
  - Files reviewed in batches when count exceeds batch-size
  - Batch processing uses separate diffs per batch for accurate file:line refs
  - Verified --help shows all focus flags

- #6: think-review: Retry logic + verbose flag - COMPLETED
  - Added --retry N flag (default 2) for retrying failed API calls
  - After max retries, prompts user "Retry again? [y/n]" 
  - Implemented in both think-review and think-tools-lib.sh call_groq
  - Added --verbose flag to show diff content before review output
  - Updated --help with new flags
  - Syntax checks pass