# wtree - Git Worktree Manager

`wtree` is a CLI tool for efficiently managing Git worktrees. Like how `ghq` manages repository clones, `wtree` provides intuitive operations for creating, switching, and deleting worktrees using a fuzzy finder interface.

![](./docs/assets/usage.gif)

## Why wtree?

Git worktrees allow you to check out multiple branches from the same repository into separate directories. This is particularly powerful when:

- Working on multiple features simultaneously
- Running parallel AI coding agents on different tasks
- Reviewing code while developing new features
- Testing changes without disrupting your main workspace

### AI Coding Agent Workflows

One of the most powerful applications of `wtree` is enabling parallel AI coding workflows. Instead of having a single AI agent work sequentially through tasks, you can leverage multiple worktrees to have multiple AI agents (like Claude Code) work on different parts of your project simultaneously:

```bash
# Create worktrees for parallel development
wtree add feature/authentication
wtree add feature/data-visualization
wtree add bugfix/login-issue

# Each AI agent can work in its own worktree
cd ~/worktrees/myapp-feature-authentication && claude
cd ~/worktrees/myapp-feature-data-visualization && claude
cd ~/worktrees/myapp-bugfix-login-issue && claude
```

Since each worktree has its own working directory with isolated files, AI agents can work at full speed without waiting for each other's changes or dealing with merge conflicts. This approach is ideal for:

- **Independent tasks**: Each AI agent focuses on a separate feature or component
- **Parallel migrations**: Multiple agents can migrate different parts of your codebase simultaneously
- **Code review workflows**: One agent writes code while another reviews it in a separate worktree
- **Testing isolation**: Run tests in one worktree while developing in another

## Installation

### Using Go
```bash
go install github.com/d-kuro/wtree/cmd/wtree@latest
```

### From Source
```bash
git clone https://github.com/d-kuro/wtree.git
cd wtree
go build -o wtree ./cmd/wtree
```

## Quick Start

```bash
# Create a new worktree with new branch
wtree add -b feature/new-ui

# List all worktrees
wtree list

# Navigate to a worktree (requires shell integration)
wtree cd

# Remove a worktree
wtree remove feature/old-ui
```

## Features

- **Fuzzy Finder Interface**: Built-in fuzzy finder for intuitive branch and worktree selection
- **Smart Navigation**: Quick switching between worktrees with pattern matching
- **Global Worktree Management**: Access all your worktrees across repositories from anywhere
- **Tab Completion**: Full shell completion support for branches, worktrees, and configuration
- **Configuration Management**: Customize worktree directories and naming conventions
- **Preview Support**: See branch details and recent commits before selection
- **Clean Operations**: Automatic cleanup of deleted worktree information
- **Branch Management**: Optional branch deletion when removing worktrees
- **Home Directory Display**: Option to display paths with `~` instead of full home directory path

## Global Worktree Management

`wtree` automatically discovers all worktrees in your configured base directory, allowing you to access them from anywhere on your system:

- **Outside Git Repositories**: When you run `wtree list` or `wtree cd` outside a git repository, it automatically discovers and shows all worktrees in the configured base directory
- **Inside Git Repositories**: By default, shows only worktrees for the current repository. Use the `-g` flag to see all worktrees from the base directory
- **Automatic Discovery**: All worktrees in the base directory are automatically discovered, including those created with native git commands
- **No Registry Required**: Uses filesystem scanning instead of maintaining a separate registry file

This feature is particularly useful when:
- Managing multiple projects simultaneously
- Quickly jumping between different repositories' worktrees
- Getting an overview of all active development branches across projects

**Note**: All worktrees located in the configured base directory (default: `~/worktrees`) are automatically discovered, regardless of how they were created.

## Commands

### `wtree add`

Create a new worktree

```bash
# Create worktree with new branch
wtree add -b feature/new-ui

# Create from existing branch
wtree add main  # Creates worktree from existing 'main' branch

# Create at specific path with new branch
wtree add -b feature/new-ui ~/projects/myapp-feature

# Create from remote branch
wtree add -b feature/api-v2 origin/feature/api-v2

# Interactive branch selection with fuzzy finder
wtree add -i
```

### `wtree list`

Display all worktrees

```bash
# Simple list
wtree list
# Output:
# BRANCH        PATH
# ● main        ~/ghq/github.com/user/project
# feature/api   ~/worktrees/github.com/user/project/feature-api
# bugfix/login  ~/worktrees/github.com/user/project/bugfix-login

# Detailed information
wtree list -v

# JSON format for scripting
wtree list --json

# Show all worktrees from base directory (from anywhere)
wtree list -g
```

### `wtree cd`

Navigate to worktree directory (requires shell integration)

```bash
# Select worktree using fuzzy finder
wtree cd

# Pattern matching
wtree cd feature

# Direct specification
wtree cd feature/new-ui

# Navigate to any worktree from base directory (from anywhere)
wtree cd -g myapp:feature
```

### `wtree remove`

Delete worktree

```bash
# Select and delete using fuzzy finder
wtree remove

# Delete by pattern
wtree remove feature/old

# Force delete
wtree remove -f feature/broken

# Delete worktree and branch together
wtree remove -b feature/completed

# Force delete branch even if not merged
wtree remove -b --force-delete-branch feature/abandoned

# Preview what would be deleted
wtree remove --dry-run -b feature/old

# Remove from any worktree in base directory (from anywhere)
wtree remove -g myapp:feature/old
```

**Branch Deletion Options:**
- By default, `wtree remove` only deletes the worktree directory, preserving the branch
- Use `-b/--delete-branch` to also delete the branch after removing the worktree
- The branch deletion uses safe mode (`git branch -d`) by default, which prevents deletion of unmerged branches
- Use `--force-delete-branch` with `-b` to force delete even unmerged branches (`git branch -D`)

### `wtree prune`

Clean up deleted worktree information

```bash
wtree prune
```

### `wtree config`

Manage configuration

```bash
# Show configuration
wtree config list

# Set worktree base directory
wtree config set worktree.basedir ~/worktrees

# Set naming template
wtree config set naming.template "{{.Repository}}-{{.Branch}}"
```

### `wtree version`

Display version information

```bash
# Show detailed version information
wtree version

# Show brief version
wtree --version
```

## Shell Integration

### Tab Completion

`wtree` provides tab completion for all commands, making it easy to discover branches, worktrees, and configuration options.

#### Setup

**Bash:**
```bash
# Add to ~/.bashrc
source <(wtree completion bash)
```

**Zsh:**
```bash
# Add to ~/.zshrc
source <(wtree completion zsh)
```

**Fish:**
```bash
# Save completion script
wtree completion fish > ~/.config/fish/completions/wtree.fish
```

**PowerShell:**
```powershell
# Add to your PowerShell profile
wtree completion powershell | Out-String | Invoke-Expression
```

After setting up, you can use tab completion:
```bash
wtree add <TAB>          # Shows available branches
wtree cd <TAB>           # Shows available worktrees
wtree remove <TAB>       # Shows branches and worktrees
wtree config set <TAB>   # Shows configuration keys
```

### Directory Navigation

The `wtree cd` command requires shell integration to actually change directories. This is because CLI tools run in a subprocess and cannot directly change the parent shell's working directory.

#### How it works

1. `wtree cd` outputs the selected worktree path instead of trying to change directories
2. A shell function captures this output and executes the actual `cd` command in your current shell

#### Setup

Add this to your `~/.bashrc` or `~/.zshrc`:

```bash
wtree() {
  case "$1" in
    cd)
      # Check if -h or --help is passed
      if [[ " ${@:2} " =~ " -h " ]] || [[ " ${@:2} " =~ " --help " ]]; then
        command wtree "$@"
      else
        local dir=$(command wtree cd --print-path "${@:2}" 2>&1)
        # Check if the command succeeded
        if [ $? -eq 0 ] && [ -n "$dir" ]; then
          cd "$dir"
        else
          # If command failed, show the error message
          echo "$dir" >&2
          return 1
        fi
      fi
      ;;
    *)
      command wtree "$@"
      ;;
  esac
}
```

After adding this function and reloading your shell (`source ~/.bashrc` or `source ~/.zshrc`), you can use `wtree cd` to navigate to worktrees:

```bash
# Interactive selection with fuzzy finder
wtree cd

# Direct navigation
wtree cd feature/new-ui
```

#### Enhanced Shell Integration for Command Chaining

If you want to use `wtree cd` with command chaining (e.g., `wtree cd && claude`), the standard shell function won't work as expected because the directory change happens after the entire command line completes.

To solve this, add this helper function to your shell configuration:

```bash
# Helper function to change to a worktree directory and run a command
wtcd() {
  local pattern=""

  # If first argument doesn't start with -, treat it as pattern
  if [ $# -gt 0 ] && [[ "$1" != -* ]]; then
    pattern="$1"
    shift
  fi

  # Get the directory path
  local dir
  if [ -n "$pattern" ]; then
    dir=$(command wtree cd --print-path "$pattern" 2>&1)
  else
    dir=$(command wtree cd --print-path 2>&1)
  fi

  # Check if wtree cd succeeded
  if [ $? -eq 0 ] && [ -n "$dir" ]; then
    cd "$dir"
    # If additional arguments provided, execute them as a command
    if [ $# -gt 0 ]; then
      "$@"
    fi
  else
    echo "$dir" >&2
    return 1
  fi
}
```

Usage examples:
```bash
# Interactive selection, then run claude
wtcd && claude

# Pattern match, then run claude
wtcd feature && claude

# Direct command execution (recommended)
wtcd feature claude

# Works with any command
wtcd api npm test
wtcd auth git status
```

<details>
<summary>Why is shell integration required?</summary>

Due to Unix/Linux process model constraints, command-line tools cannot directly change the parent shell's working directory:

- When you run `wtree cd`, it creates a new process
- Even if that process calls `chdir()`, it only affects its own process
- When the process exits, the shell remains in the original directory

This is a security feature - child processes cannot modify parent process state. Other popular tools like `z`, `fasd`, `autojump`, and `fzf` use the same shell function approach.

Alternative usage without shell integration:
```bash
# Get path and cd manually
cd $(wtree cd --print-path feature/new-ui)

# Start new shell in target directory
wtree cd feature/new-ui && bash
```
</details>

## Configuration

Configuration file location: `~/.config/wtree/config.toml`

```toml
[worktree]
# Base directory for creating worktrees
basedir = "~/worktrees"
# Automatically create directories
auto_mkdir = true

[finder]
# Enable preview window
preview = true
# Preview window size
preview_size = 3

[naming]
# Directory name template
# Available variables: Host, Owner, Repository, Branch, Hash
template = "{{.Host}}/{{.Owner}}/{{.Repository}}/{{.Branch}}"
# Invalid character replacement (applied to branch names)
sanitize_chars = { "/" = "-", ":" = "-" }

[ui]
# Color output
color = true
# Icon display
icons = true
# Display home directory as ~ in paths
tilde_home = true
```

## Advanced Usage

### Multiple AI Agent Workflow

```bash
# Create multiple worktrees for parallel development
wtree add -b feature/auth
wtree add -b feature/api
wtree add -b bugfix/login

# Launch AI agents in parallel (example with Claude Code)
# Without shell integration:
# Terminal 1
cd $(wtree cd --print-path auth) && claude

# Terminal 2
cd $(wtree cd --print-path api) && claude

# Terminal 3
cd $(wtree cd --print-path login) && claude

# With shell integration enabled (simpler):
# Terminal 1
wtree cd auth && claude

# Terminal 2
wtree cd api && claude

# Terminal 3
wtree cd login && claude
```

### Batch Operations

```bash
# List all feature branches from global worktrees
wtree list -g --json | jq '.[] | select(.branch | contains("feature"))'

# Clean up old feature worktrees
wtree list -g --json | \
  jq -r '.[] | select(.branch | contains("feature/old-")) | .branch' | \
  xargs -I {} wtree remove -g {}
```

### Integration with Git Workflows

```bash
# Create worktree for PR review
wtree add -b pr-123-review origin/pull/123/head

# Create worktree for hotfix
wtree add -b hotfix/critical-bug origin/main

# Switch between worktrees quickly
wtree cd  # Use fuzzy finder to select
```

### Version Information

```bash
# Show version information
wtree version

# Show brief version
wtree --version
```

## Directory Structure

`wtree` organizes worktrees using a URL-based hierarchy similar to `ghq`, ensuring no naming conflicts:

```
~/worktrees/
├── github.com/
│   ├── user1/
│   │   └── myapp/
│   │       ├── main/           # Main branch
│   │       ├── feature-auth/   # Authentication feature
│   │       └── feature-api/    # API development
│   └── user2/
│       └── myapp/              # Same repo name, different user
│           ├── main/
│           └── develop/
├── gitlab.com/
│   └── company/
│       └── project/
│           └── feature-x/
└── code.google.com/
    └── p/
        └── vim/
            └── main/
```

This structure:
- **Prevents conflicts**: Same repository names from different sources don't collide
- **Preserves context**: You always know which repository a worktree belongs to
- **Scales naturally**: Works with any number of git hosting services
- **Follows conventions**: Similar to how `ghq` manages repository clones

## Requirements

- Git 2.5+ (for worktree support)
- Go 1.24+ (for building from source)
- Terminal with Unicode support (for fuzzy finder)

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

Apache License 2.0 - see [LICENSE](LICENSE) file for details.
