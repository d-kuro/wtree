# gwq Project Guidelines

## Bash Commands
- `make build`: Build the project
- `make test`: Run tests
- `make lint`: Run linters
- `go test ./...`: Run all tests
- `go mod tidy`: Clean up module dependencies

## Code Style
- Follow standard Go conventions and idioms
- Use `goimports` for code formatting
- Follow existing patterns in the codebase
- Keep functions focused and testable
- Use meaningful variable and function names
- IMPORTANT: Always check existing code patterns before implementing new features

## Testing Instructions
- Write tests for new functionality
- Run tests before committing: `go test ./...`
- Prefer table-driven tests for multiple test cases
- Mock external dependencies appropriately
- Ensure test coverage for edge cases

## Workflow Guidelines
- IMPORTANT: Always run `make lint` and `make test` after making code changes
- Research existing code patterns before implementing new features
- When editing code, first understand the surrounding context and imports
- Be specific and thorough when planning complex changes
- Create a todo list for multi-step tasks
- Verify all changes compile and pass tests before marking tasks complete

### Complex Workflow Management
- For large tasks with multiple steps (e.g., fixing many lint errors, large refactorings):
  1. Run the relevant command (e.g., `make lint`) and write all errors to a Markdown checklist
  2. Address each issue one by one, fixing and verifying before checking it off
  3. Use a scratchpad file to track progress and maintain context
- This systematic approach ensures no issues are missed and maintains clarity throughout the task

## Git/GitHub Conventions
- Write clear, concise commit messages explaining the "why" not just the "what"
- Follow existing commit message style in the repository
- Create descriptive PR titles and descriptions
- Reference issue numbers in commits when applicable

### GitHub Operations
- Use `gh` CLI for GitHub interactions:
  - `gh pr create`: Create pull requests
  - `gh pr list`: List pull requests
  - `gh issue create`: Create issues
  - `gh issue list`: List issues
  - `gh pr comment`: Add comments to PRs
  - `gh pr review`: Review pull requests
- Claude can use `gh` to automate GitHub workflows like creating PRs, fixing review comments, and triaging issues
- Install `gh` CLI if not already available: https://cli.github.com/

## Important Reminders
- NEVER write code without understanding the existing patterns first
- ALWAYS verify your changes with linting and testing
- DO NOT make assumptions about available libraries - check go.mod first
- When searching for code patterns, examine similar files in the codebase
- Follow security best practices - never expose or log secrets

## New Features Added
- **Tmux Session Management**: The `gwq tmux` subcommand group provides persistence for long-running processes
  - `gwq tmux list`: List active tmux sessions
  - `gwq tmux run`: Create new tmux sessions with commands
  - `gwq tmux attach`: Attach to running sessions
  - `gwq tmux kill`: Terminate sessions
- Sessions are managed in the `internal/tmux` package
- Fuzzy finder integration available for session selection
- Context cancellation is properly implemented for long-running operations