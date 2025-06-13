# gwq Release Notes

## v0.0.4-rc.1 (2025-06-13)

> ⚠️ **RELEASE CANDIDATE WARNING**  
> This is a release candidate and may contain breaking changes. Features and APIs are subject to change without notice. Not recommended for production use.

### 🚀 New Features

#### Tmux Session Management (#11)
- **New `gwq tmux` command group** for comprehensive session management
  - `gwq tmux list` - List active tmux sessions with JSON/CSV output support
  - `gwq tmux run` - Create new tmux sessions for long-running processes
  - `gwq tmux attach` - Attach to running sessions with fuzzy finder
  - `gwq tmux kill` - Terminate sessions with batch operations
- **Persistent session support** for development servers, builds, and tests
- **Real-time monitoring** and interactive session selection

#### Claude Task Queue System (#12)
- **Complete task management CLI** with YAML-based configuration
  - `gwq task add` - Add tasks individually or from YAML files
  - `gwq task list` - List all tasks with status and priority filtering
  - `gwq task worker start` - Start task execution worker with parallel control
  - `gwq task worker stop` - Stop running workers
  - `gwq task logs` - View task-specific execution logs
  - `gwq task status` - Check task execution status and dependencies
- **Advanced dependency management** with graph-based resolution and cycle detection
- **Priority-based scheduling** (0-100 scale) with parallel execution control
- **Automatic Git worktree management** for isolated task execution
- **Comprehensive logging** with JSON-structured, task-specific logs
- **tmux integration** for persistent task sessions

### 🔧 Technical Improvements

- Added `internal/tmux` package for session lifecycle management
- Implemented extensible, agent-based architecture for task execution
- Enhanced context cancellation support
- Simplified configuration structure
- Removed unused code and dependencies

### 📋 Sample Task File (tasks.yaml)

```yaml
version: "1.0"
repository: /path/to/your/project  # REQUIRED: Absolute path only

# Default configuration for all tasks
default_config:
  skip_permissions: true
  timeout: "2h"
  max_iterations: 3
  dependency_policy: "wait"
  priority: 50

tasks:
  - id: lint-check
    worktree: release/lint-check
    base_branch: main
    priority: 90
    prompt: "Run make lint and fix any issues found"

  - id: test-suite
    worktree: release/test-suite
    base_branch: main
    priority: 85
    prompt: "Execute make test and ensure all tests pass"
    depends_on: ["lint-check"]

  - id: build-check
    worktree: release/build-check
    base_branch: main
    priority: 80
    prompt: "Run make build and verify successful compilation"
    depends_on: ["test-suite"]

  - id: docs-update
    worktree: release/docs-update
    base_branch: main
    priority: 60
    prompt: "Update documentation and README if needed"

  - id: docs-update
    worktree: release/docs-update
    base_branch: main
    priority: 95
    prompt: "Prepare release v0.0.4-rc.1 and update version files"
    depends_on: ["build-check", "docs-update"]
```

#### Task Management Examples

Adding tasks:

```console
$ gwq task add claude -f tasks.yaml
Task 'Run make lint and fix any issues found' (f1a2b3c4) added successfully
Repository: /path/to/your/project
Worktree: release/lint-check, Priority: 90

Task 'Execute make test and ensure all tests pass' (d5e6f7g8) added successfully
Repository: /path/to/your/project
Worktree: release/test-suite, Priority: 85
Dependencies: f1a2b3c4

Successfully added 5 tasks from tasks.yaml
```

Starting worker:

```console
$ gwq task worker start --parallel 2
Starting Claude Code worker (max parallel: 2)
Worker started, polling for tasks...
Starting task: Run make lint and fix any issues found (ID: f1a2b3c4)
Starting task: Update documentation and README if needed (ID: l3m4n5o6)
Creating worktree 'release/lint-check' from base branch 'main'...
Creating worktree 'release/docs-update' from base branch 'main'...
Task completed: l3m4n5o6
Task completed: f1a2b3c4
Starting task: Execute make test and ensure all tests pass (ID: d5e6f7g8)
```

### 📖 Usage Examples

#### Task Queue Management
```bash
# Add single task
gwq task add claude "Fix linting issues in main package"

# Add tasks from YAML file
gwq task add claude -f tasks.yaml

# List all tasks
gwq task list

# List tasks with filtering
gwq task list --status pending --priority high

# Start worker with parallel execution
gwq task worker start --parallel 2

# Stop worker
gwq task worker stop

# View task logs
gwq task logs <task-id>

# Check task status
gwq task status <task-id>
```

#### Tmux Session Management
```bash
# List active sessions
gwq tmux list --format json

# Create and run session
gwq tmux run --session dev-server "npm run dev"

# Attach to session
gwq tmux attach

# Kill specific session
gwq tmux kill --session dev-server
```

### ⚠️ Breaking Changes

- New command structure may conflict with existing workflows
- Configuration format changes for task management
- API changes in internal packages

### 🐛 Known Issues

- Task dependency resolution may have edge cases
- tmux integration requires tmux to be installed
- Some error handling may need refinement

---

## v0.0.4 (Unreleased)

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
