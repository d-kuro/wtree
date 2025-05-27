# gwq - Git Worktree Manager Design

## Overview

`gwq` is a CLI tool for efficiently managing Git worktrees with global discovery capabilities. It follows the same organizational principles as `ghq` for repository cloning, providing a structured approach to worktree management across multiple repositories and hosting services.

## Core Principles

### 1. Global Worktree Management

- **Filesystem-based Discovery**: Automatically discovers all worktrees in the configured base directory
- **No Registry Files**: Uses filesystem scanning instead of maintaining separate registry files
- **Cross-Repository Operations**: Manage worktrees across multiple repositories from anywhere

### 2. URL-Based Hierarchy

- **Conflict Prevention**: Uses repository URL hierarchy to prevent naming conflicts
- **Scalable Structure**: Supports unlimited repositories and hosting services
- **Intuitive Organization**: Follows familiar patterns from tools like `ghq`

### 3. User Experience

- **Shell Integration**: Native directory navigation through shell functions
- **Fuzzy Finding**: Interactive selection with preview capabilities
- **Pattern Matching**: Flexible worktree identification and selection
- **Tab Completion**: Full shell completion support for enhanced discoverability

## Architecture

### Directory Structure

```
~/worktrees/
├── github.com/
│   ├── user1/
│   │   └── myapp/
│   │       ├── main/
│   │       ├── feature-auth/
│   │       └── feature-api/
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

### Benefits of URL Hierarchy

- **No Name Conflicts**: Different repositories with same names coexist safely
- **Clear Context**: Always know which repository a worktree belongs to
- **Natural Scaling**: Works with any number of git hosting services
- **Familiar Pattern**: Consistent with `ghq` and other development tools

## Command Design

### Core Commands

#### `gwq add [options] <branch> [<path>]`

Create new worktrees with automatic URL-based path generation

- Supports both new and existing branches
- Interactive branch selection with fuzzy finder (`-i` flag)
- Custom path specification when needed
- Remote branch support

#### `gwq list [options]`

Display worktrees with context-aware behavior

- **Inside Repository**: Shows local worktrees by default
- **Outside Repository**: Shows all discovered worktrees
- **Global Flag**: Always shows all worktrees regardless of location
- **Output Formats**: Table (default), verbose (`-v`), JSON (`--json`)
- Shows current worktree with bullet indicator

#### `gwq cd [pattern]`

Navigate to worktree directories with shell integration

- Fuzzy finder for interactive selection
- Pattern matching for quick navigation
- Global mode for cross-repository navigation
- Repository prefix support (e.g., `myapp:feature`)

#### `gwq remove [pattern]`

Delete worktrees with safety features and optional branch deletion

- Interactive selection and confirmation
- Pattern matching for batch operations
- Dry-run mode for safety (`--dry-run`)
- Optional branch deletion with `-b/--delete-branch`
- Safe deletion by default, force deletion with `--force-delete-branch`
- Multiple selection support in interactive mode

#### `gwq prune`

Clean up stale worktree information

- Removes administrative files for deleted worktrees
- Handles manually deleted directories
- No effect on properly removed worktrees

#### `gwq config`

Manage configuration settings

- View current configuration (`gwq config list`)
- Set configuration values (`gwq config set <key> <value>`)
- Hierarchical key support (e.g., `worktree.basedir`)

#### `gwq version`

Display version information

- Detailed version with build information
- Brief version with `--version` flag

### Global Operation Modes

All primary commands support dual modes:

1. **Local Mode** (inside repository): Operates on current repository's worktrees
2. **Global Mode** (outside repository or `-g` flag): Operates on all discovered worktrees

## Technical Architecture

### Package Structure

```
gwq/
├── cmd/gwq/              # Main entry point
├── internal/
│   ├── cmd/               # Command implementations
│   ├── config/           # Configuration management
│   ├── discovery/        # Filesystem-based worktree discovery
│   ├── finder/           # Fuzzy finder integration
│   ├── git/              # Git operations wrapper
│   ├── registry/         # Worktree registry (deprecated)
│   ├── ui/               # User interface components
│   ├── url/              # Repository URL parsing and hierarchy
│   └── worktree/         # Worktree management logic
└── pkg/
    ├── cache/            # Caching utilities
    ├── models/           # Data structures
    ├── option/           # Option types and utilities
    ├── pipeline/         # Pipeline processing utilities
    ├── repository/       # Repository information handling
    ├── result/           # Result type utilities
    └── utils/            # General utilities
```

### Key Components

#### URL Parser (`internal/url/`)

- Parses various git URL formats (SSH, HTTPS, etc.)
- Extracts host, owner, and repository information
- Generates hierarchical paths for worktree placement
- Handles branch name sanitization for filesystem compatibility

#### Discovery System (`internal/discovery/`)

- Scans configured base directory for worktrees
- Identifies git worktrees by `.git` file presence
- Extracts repository and branch information
- No registry maintenance required

#### Configuration Management (`internal/config/`)

- TOML-based configuration with sensible defaults
- Template-based path generation (maintained for backward compatibility)
- UI and finder customization options
- Supports color, icons, and tilde home display preferences

#### Completion System (`internal/cmd/completion.go`)

- Provides shell completion for all commands
- Dynamic completion based on current repository state
- Supports branches, worktrees, and configuration keys
- Context-aware completions (local vs global mode)

### Tab Completion

Tab completion is supported for all major shells:

- **Bash**: `source <(gwq completion bash)`
- **Zsh**: `source <(gwq completion zsh)`
- **Fish**: `gwq completion fish > ~/.config/fish/completions/gwq.fish`
- **PowerShell**: `gwq completion powershell | Out-String | Invoke-Expression`

Completion features:
- Branch names for `add` and `remove` commands
- Worktree names for `cd` and `remove` commands
- Configuration keys for `config set` command
- Flag completions for all commands

## Configuration

### Default Configuration

```toml
[worktree]
basedir = "~/worktrees"
auto_mkdir = true

[finder]
preview = true
preview_size = 3

[naming]
# Template for directory names (optional, URL hierarchy is preferred)
template = "{{.Host}}/{{.Owner}}/{{.Repository}}/{{.Branch}}"
# Character replacements for filesystem compatibility
sanitize_chars = { "/" = "-", ":" = "-" }

[ui]
color = true
icons = true
tilde_home = true  # Display ~ instead of full home path
```

### Configuration Management

- Template-based naming is maintained for backward compatibility
- URL hierarchy is the recommended approach for new installations
- Character sanitization ensures filesystem compatibility across platforms

## Shell Integration

### Technical Requirement

Shell integration is necessary because CLI tools run in child processes and cannot directly change the parent shell's working directory due to Unix process isolation.

### Implementation

```bash
gwq() {
  case "$1" in
    cd)
      # Check if -h or --help is passed
      if [[ " ${@:2} " =~ " -h " ]] || [[ " ${@:2} " =~ " --help " ]]; then
        command gwq "$@"
      else
        local dir=$(command gwq cd --print-path "${@:2}" 2>&1)
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
      command gwq "$@"
      ;;
  esac
}
```

This approach is consistent with other popular tools like `z`, `fasd`, `autojump`, and `fzf`.

### Enhanced Shell Integration for Command Chaining

The standard shell function doesn't support command chaining (e.g., `gwq cd && claude`) because the directory change happens after the entire command line completes. To address this, we provide an enhanced helper function:

```bash
gwcd() {
  local pattern=""
  
  # If first argument doesn't start with -, treat it as pattern
  if [ $# -gt 0 ] && [[ "$1" != -* ]]; then
    pattern="$1"
    shift
  fi
  
  # Get the directory path
  local dir
  if [ -n "$pattern" ]; then
    dir=$(command gwq cd --print-path "$pattern" 2>&1)
  else
    dir=$(command gwq cd --print-path 2>&1)
  fi
  
  # Check if gwq cd succeeded
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

### Error Handling

The shell functions include proper error handling to ensure:
- Help flags are passed through correctly
- Error messages are displayed to the user
- Failed commands don't attempt directory changes
- Proper exit codes are returned

## Use Cases

### Multi-Repository Development

- Work on multiple repositories simultaneously
- Quick context switching between projects
- Consistent worktree organization across all repositories

### AI-Assisted Development

- Parallel AI coding agents working on different features
- Isolated development environments for each agent
- No conflicts between simultaneous operations

### Team Collaboration

- Standardized worktree organization across team members
- Easy sharing of worktree-based workflows
- Consistent repository structure regardless of hosting service

## Design Decisions

### Filesystem over Registry

**Decision**: Use filesystem scanning instead of registry files
**Rationale**:

- Eliminates sync issues between registry and actual filesystem state
- Works with manually created worktrees
- Simpler, more reliable architecture
- No maintenance overhead

### URL Hierarchy over Templates

**Decision**: Use repository URL-based hierarchy instead of configurable templates
**Rationale**:

- Prevents all naming conflicts
- Provides consistent, predictable structure
- Reduces configuration complexity
- Follows established patterns from `ghq`

### Global vs Local Context

**Decision**: Automatic context detection with explicit override
**Rationale**:

- Intuitive behavior for most common use cases
- Maintains local repository focus when working within one
- Easy access to global operations when needed
- Consistent with user expectations

### Branch Deletion in Remove Command

**Decision**: Branch deletion is opt-in, not default
**Rationale**:

- **Data Safety**: Branches contain commit history that may not be merged
- **Git Philosophy**: Worktrees and branches are independent concepts
- **Backward Compatibility**: Preserves expected behavior for existing users
- **Flexibility**: Same branch can have multiple worktrees

**Implementation**:
- `-b/--delete-branch`: Enables branch deletion after worktree removal
- Uses safe deletion (`git branch -d`) by default
- `--force-delete-branch`: Force deletion (`git branch -D`) for unmerged branches
- Clear success messages for both worktree and branch operations

## Future Considerations

### Extensibility

- Plugin architecture for custom worktree management workflows
- Integration with IDEs and editors
- Support for additional VCS systems

### Performance

- Lazy loading for large worktree collections
- Caching strategies for repository information
- Parallel discovery operations

### Compatibility

- Windows support considerations
- Integration with existing Git workflows
- Backward compatibility for configuration migration

## License

Apache License 2.0
