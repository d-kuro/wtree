# gwq Release Notes

## v0.0.3 (2025-06-09)

### ✨ New Features
- **Comprehensive Worktree Status Dashboard** (#7): Monitor all worktrees at a glance with real-time visibility into git status, changes, and activity
  - **Multiple Output Formats**: Table (default), JSON, and CSV formats for integration with other tools
  - **Watch Mode**: Auto-refresh with configurable intervals (`--watch`) for real-time monitoring
  - **Advanced Filtering & Sorting**: Filter by status and sort by various fields (branch, activity, modifications)
  - **AI Agent Monitoring**: Perfect for tracking multiple AI coding agents working across different worktrees
  - **Process Information**: Optional process monitoring to see which tools are active in each worktree
  - **Activity Tracking**: Show last modification time and recent activity for each worktree
  - **Parallel Collection**: Efficient concurrent status gathering for fast performance

### 📊 Status Command Examples
```bash
# Basic status view
gwq status

# Watch mode for real-time monitoring  
gwq status --watch

# JSON output for scripting
gwq status --json | jq '.worktrees[] | select(.status == "changed")'

# Filter and export to CSV
gwq status --filter changed --csv > worktree-report.csv
```

### 🤖 AI Development Workflow Enhancement
- Enhanced README with AI agent monitoring examples
- Real-time visibility into which agents have made changes and when
- Progress tracking across multiple parallel development efforts
- Integration examples for batch operations and reporting

### 📖 Documentation
- Added comprehensive design document (`docs/DESIGN_STATUS_DASHBOARD.md`)
- Updated README with status command usage and AI workflow examples
- Expanded configuration documentation

### 🧪 Testing
- Comprehensive test coverage for status functionality
- Table-driven tests for all formatting functions
- Performance testing with multiple worktrees

**Full Changelog**: [v0.0.2...v0.0.3](https://github.com/d-kuro/gwq/compare/v0.0.2...v0.0.3)

---

## v0.0.2 (2025-05-31)

### 🐛 Bug Fixes
- Fixed URL normalization for ssh:// prefixed Git URLs (#6 by @osamu2001)

### 👥 New Contributors
- @osamu2001 made their first contribution in #6

**Full Changelog**: [v0.0.1...v0.0.2](https://github.com/d-kuro/gwq/compare/v0.0.1...v0.0.2)

---

# gwq v0.0.1 Release Notes

🎉 **Initial Release**

We're excited to announce the first release of gwq - a CLI tool for efficient Git worktree management with global discovery capabilities.

> ⚠️ **Experimental Release**: This is an early experimental version. Breaking changes may occur in future releases as we refine the API and features based on user feedback.

## 🌟 Features

### Core Functionality
- **Global Worktree Discovery**: Automatically find and manage worktrees across all repositories in your configured base directory
- **URL-Based Organization**: Prevent naming conflicts using repository URL hierarchy (e.g., `~/worktrees/github.com/user/repo/branch`)
- **Fuzzy Finder Interface**: Interactive selection with pattern matching, preview support, and multiple selection capabilities
- **AI-Powered Workflows**: Enable parallel development with multiple AI coding agents working on different features simultaneously

### Commands
- `gwq add` - Create new worktrees with automatic path generation
- `gwq list` - Display all worktrees with context-aware behavior
- `gwq get` - Retrieve worktree path for shell substitution
- `gwq exec` - Execute commands in worktree directory
- `gwq remove` - Delete worktrees with optional branch deletion
- `gwq prune` - Clean up stale worktree information
- `gwq config` - Manage configuration settings
- `gwq completion` - Generate shell completions
- `gwq version` - Display version information

### Shell Integration
- Full tab completion support for Bash, Zsh, Fish, and PowerShell
- Quick navigation with `cd $(gwq get <worktree>)`
- Execute commands without changing directory using `gwq exec`

### Configuration
Customize behavior through `~/.config/gwq/config.toml`:
- Base directory for worktrees
- Fuzzy finder preview settings
- UI preferences (colors, icons, display options)

## 📋 Requirements
- Git 2.5 or higher
- Go 1.24+ (for building from source)

## 📦 Installation

```bash
go install github.com/d-kuro/gwq/cmd/gwq@v0.0.1
```

## 🤝 Contributing
As this is an experimental release, we welcome feedback, bug reports, and feature requests! Please open issues on our [GitHub repository](https://github.com/d-kuro/gwq).

## ⚠️ Note
This is an experimental v0.0.1 release. The API and behavior may change significantly in future versions as we iterate based on user feedback and requirements. Please be prepared for potential breaking changes in subsequent releases.

---

For detailed documentation and usage examples, visit our [GitHub repository](https://github.com/d-kuro/gwq).