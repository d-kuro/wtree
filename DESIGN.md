# wtree - Git Worktree Manager Design

## Overview

`wtree` is a CLI tool for efficiently managing Git worktrees with global discovery capabilities. It follows the same organizational principles as `ghq` for repository cloning, providing a structured approach to worktree management across multiple repositories and hosting services.

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

#### `wtree add [options] <branch> [<path>]`

Create new worktrees with automatic URL-based path generation

- Supports both new and existing branches
- Interactive branch selection with fuzzy finder
- Custom path specification when needed

#### `wtree list [options]`

Display worktrees with context-aware behavior

- **Inside Repository**: Shows local worktrees by default
- **Outside Repository**: Shows all discovered worktrees
- **Global Flag**: Always shows all worktrees regardless of location

#### `wtree cd [pattern]`

Navigate to worktree directories with shell integration

- Fuzzy finder for interactive selection
- Pattern matching for quick navigation
- Global mode for cross-repository navigation

#### `wtree remove [pattern]`

Delete worktrees with safety features

- Interactive selection and confirmation
- Pattern matching for batch operations
- Dry-run mode for safety

### Global Operation Modes

All primary commands support dual modes:

1. **Local Mode** (inside repository): Operates on current repository's worktrees
2. **Global Mode** (outside repository or `-g` flag): Operates on all discovered worktrees

## Technical Architecture

### Package Structure

```
wtree/
├── cmd/wtree/              # Main entry point
├── internal/
│   ├── cmd/               # Command implementations
│   ├── config/           # Configuration management
│   ├── discovery/        # Filesystem-based worktree discovery
│   ├── finder/           # Fuzzy finder integration
│   ├── git/              # Git operations wrapper
│   ├── ui/               # User interface components
│   ├── url/              # Repository URL parsing and hierarchy
│   └── worktree/         # Worktree management logic
└── pkg/
    └── models/           # Data structures
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
- Template-based path generation (deprecated in favor of URL hierarchy)
- UI and finder customization options

## Configuration

### Default Configuration

```toml
[worktree]
basedir = "~/worktrees"
auto_mkdir = true

[finder]
preview = true
preview_size = 3

[ui]
color = true
icons = true
```

### URL-Based Path Generation

Paths are automatically generated using repository URL hierarchy, replacing the previous template-based system for better conflict prevention and consistency.

## Shell Integration

### Technical Requirement

Shell integration is necessary because CLI tools run in child processes and cannot directly change the parent shell's working directory due to Unix process isolation.

### Implementation

```bash
wtree() {
  case "$1" in
    cd)
      local dir=$(command wtree cd --print-path "${@:2}")
      if [ -n "$dir" ]; then
        cd "$dir"
      fi
      ;;
    *)
      command wtree "$@"
      ;;
  esac
}
```

This approach is consistent with other popular tools like `z`, `fasd`, `autojump`, and `fzf`.

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
