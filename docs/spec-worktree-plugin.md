# Worktree Manager Plugin Technical Specification

> A sidecar plugin for orchestrating git worktrees with AI coding agents

**Version:** 0.1.0-draft  
**Status:** Design Phase  
**Authors:** Collaborative design session  
**Last Updated:** January 2026

---

## Table of Contents

1. [Overview](#1-overview)
2. [Goals and Non-Goals](#2-goals-and-non-goals)
3. [Architecture](#3-architecture)
4. [User Interface Design](#4-user-interface-design)
5. [Git Worktree Operations](#5-git-worktree-operations)
6. [tmux Integration](#6-tmux-integration)
7. [Agent Status Detection](#7-agent-status-detection)
8. [TD Task Manager Integration](#8-td-task-manager-integration)
9. [Data Persistence](#9-data-persistence)
10. [Configuration](#10-configuration)
11. [Safety Precautions](#11-safety-precautions)
12. [Implementation Phases](#12-implementation-phases)
13. [Reference Implementations](#13-reference-implementations)
14. [Appendix: Command Reference](#appendix-command-reference)

---

## 1. Overview

### 1.1 Problem Statement

Developers using AI coding agents (Claude Code, Codex, Aider, Gemini) want to run multiple agents in parallel on different features without:

- Branch conflicts from concurrent work
- Losing visibility into what each agent is doing
- Manual context-switching overhead
- Forgetting to commit/push completed work

### 1.2 Solution

A sidecar plugin that:

- Manages git worktrees for isolated parallel development
- Runs agents in tmux sessions for process isolation
- Provides a unified TUI to monitor all agents
- Integrates with `td` for task tracking and handoffs

### 1.3 Prior Art

| Tool                                                                      | Strengths                                   | Gaps                           |
| ------------------------------------------------------------------------- | ------------------------------------------- | ------------------------------ |
| [Claude Squad](https://github.com/smtg-ai/claude-squad)                   | Proven tmux/worktree pattern, BubbleTea TUI | No task management integration |
| [Conductor](https://conductor.ai)                                         | Beautiful dashboard, auto-management        | macOS GUI only, closed source  |
| [Treehouse Worktree](https://github.com/mark-hingston/treehouse-worktree) | MCP support, lock system                    | No TUI, focused on Cursor      |

This plugin bridges the gap by combining worktree orchestration with td's task management in a terminal-native interface that fits sidecar's existing plugin architecture.

---

## 2. Goals and Non-Goals

### 2.1 Goals

- **G1**: View all worktrees and their agent status at a glance
- **G2**: Create worktrees linked to td tasks with one command
- **G3**: See live agent output without leaving sidecar
- **G4**: Approve/reject agent prompts from the TUI
- **G5**: Push, merge, and cleanup completed work
- **G6**: Resume crashed agents on existing worktrees

### 2.2 Non-Goals

- **NG1**: Replace the agent's native UI (users can always attach)
- **NG2**: Implement a full IDE (no editing, just viewing)
- **NG3**: Support non-git version control systems
- **NG4**: Cross-machine synchronization

---

## 3. Architecture

### 3.1 High-Level Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                         sidecar                                 │
│  ┌──────────┬──────────┬──────────┬──────────┬──────────────┐   │
│  │   Git    │  Files   │    TD    │  Convos  │  Worktrees   │   │
│  │  Plugin  │  Plugin  │  Plugin  │  Plugin  │   Plugin     │   │
│  └──────────┴──────────┴──────────┴──────────┴──────┬───────┘   │
└─────────────────────────────────────────────────────┼───────────┘
                                                      │
              ┌───────────────────────────────────────┼──────────┐
              │            Worktree Manager           │          │
              │  ┌─────────────────────────────────────────────┐ │
              │  │              WorktreeManager                │ │
              │  │  - worktrees: []*Worktree                   │ │
              │  │  - agents: map[string]*Agent                │ │
              │  │  - tdClient: *TDClient                      │ │
              │  └──────────────┬─────────────┬───────────────┘ │
              │                 │             │                  │
              │    ┌────────────┴─┐  ┌────────┴────────┐        │
              │    │  Git Ops     │  │  tmux Manager   │        │
              │    │  - add       │  │  - sessions     │        │
              │    │  - remove    │  │  - capture      │        │
              │    │  - list      │  │  - send-keys    │        │
              │    └──────────────┘  └─────────────────┘        │
              └──────────────────────────────────────────────────┘
                        │                     │
                        ▼                     ▼
              ┌─────────────────┐   ┌─────────────────────┐
              │   Git Repo      │   │   tmux Server       │
              │  .git/worktrees │   │  sidecar-wt-*       │
              └─────────────────┘   └─────────────────────┘
```

### 3.2 Package Structure

```
internal/plugins/worktree/
├── plugin.go           # Plugin interface implementation
├── model.go            # BubbleTea model and state
├── view.go             # UI rendering (list view, preview pane)
├── view_kanban.go      # Kanban view
├── keymap.go           # Keyboard shortcut definitions
├── worktree.go         # Git worktree operations
├── agent.go            # Agent process management
├── tmux.go             # tmux session management
├── td.go               # TD integration
├── config.go           # Configuration handling
├── status.go           # Status detection logic
└── types.go            # Shared types and constants
```

### 3.3 Core Types

```go
// Worktree represents a git worktree with optional agent
type Worktree struct {
    Name       string          // e.g., "auth-oauth-flow"
    Path       string          // absolute path
    Branch     string          // git branch name
    BaseBranch string          // branch worktree was created from
    TaskID     string          // linked td task (e.g., "td-a1b2")
    Agent      *Agent          // nil if no agent running
    Status     WorktreeStatus  // derived from agent state
    Stats      *GitStats       // +/- line counts
    CreatedAt  time.Time
    UpdatedAt  time.Time
}

type WorktreeStatus int

const (
    StatusPaused   WorktreeStatus = iota // No agent, worktree exists
    StatusActive                         // Agent running, recent output
    StatusWaiting                        // Agent waiting for input
    StatusDone                           // Agent completed task
    StatusError                          // Agent crashed or errored
)

// Agent represents an AI coding agent process
type Agent struct {
    Type        AgentType       // claude, codex, aider, gemini
    TmuxSession string          // tmux session name
    TmuxPane    string          // pane identifier
    PID         int             // process ID (if available)
    StartedAt   time.Time
    LastOutput  time.Time       // last time output was detected
    OutputBuf   *ring.Buffer    // last N lines of output
    Status      AgentStatus
    WaitingFor  string          // prompt text if waiting
}

type AgentType string

const (
    AgentClaude AgentType = "claude"
    AgentCodex  AgentType = "codex"
    AgentAider  AgentType = "aider"
    AgentGemini AgentType = "gemini"
    AgentCustom AgentType = "custom"
)

// GitStats holds file change statistics
type GitStats struct {
    Additions    int
    Deletions    int
    FilesChanged int
    Ahead        int  // commits ahead of base branch
    Behind       int  // commits behind base branch
}
```

---

## 4. User Interface Design

### 4.1 List View (Default)

The primary view shows worktrees in a split-pane layout:

```
┌─────────────────────────────────────────────────────────────────────────────┐
│ [1]Git [2]Files [3]TD [4]Convos [5]Worktrees                                │
├────────────────────────────────────────┬────────────────────────────────────┤
│ Worktrees                    [List|Kan]│ Output                  Diff  Task │
├────────────────────────────────────────┼────────────────────────────────────┤
│ ● auth-oauth-flow           💬  3m    │ Claude I'll implement the OAuth    │
│   claude  td-a1b2    +47 -12          │ 2.0 callback handler for your      │
│                                        │ authentication flow.               │
│ ○ payment-refactor          🟢  18m   │                                    │
│   codex   td-c3d4    +156 -34         │ Reading internal/auth/handler.go   │
│                                        │ Reading internal/config/oauth.go   │
│ ○ hotfix-login-timeout      ✅  42m   │                                    │
│   claude  td-e5f6    +8 -2            │ Claude I see the existing auth     │
│                                        │ structure. I'll add the OAuth      │
│ ○ ui-redesign-nav           ⏸   2h    │ callback endpoint that:            │
│   —       td-g7h8    +0 -0            │  1. Validates the state parameter  │
│                                        │  2. Exchanges the auth code        │
│                                        │  3. Creates the user session       │
│                                        │                                    │
│                                        │ Creating oauth_callback.go         │
│                                        │ Modifying routes.go                │
│                                        │                                    │
│                                        │ ┌──────────────────────────────┐   │
│                                        │ │ Allow edit to oauth_callback │   │
│                                        │ │ .go? (new file, 47 lines)    │   │
│                                        │ │                              │   │
│                                        │ │   [y] yes  [n] no  [e] view  │   │
│                                        │ └──────────────────────────────┘   │
│                                        │                                    │
├────────────────────────────────────────┴────────────────────────────────────┤
│ n:new  y:approve  ↵:attach  d:diff  p:push  m:merge  D:delete  ?:help       │
│                                               🟢 2 active  💬 1 waiting 17:42│
└─────────────────────────────────────────────────────────────────────────────┘
```

**Status Indicators:**

- `🟢` / `●` Active (green) - Agent running, recent output
- `💬` / `○` Waiting (yellow, pulsing) - Agent needs input
- `✅` Done (cyan) - Agent completed or printed completion message
- `⏸` Paused (gray) - No agent running
- `❌` Error (red) - Agent crashed

### 4.2 Kanban View

Toggle with `v` key for column-based organization:

```
┌─────────────────────────────────────────────────────────────────────────────┐
│ [1]Git [2]Files [3]TD [4]Convos [5]Worktrees                                │
├────────────────────────────────────────────────────────────────────────────┤
│ Worktrees                                                      [List|Kanban]│
├─────────────────┬─────────────────┬─────────────────┬─────────────────────┤
│ 🟢 Active (2)   │ 💬 Waiting (1)  │ ✅ Ready (2)    │ ⏸ Paused (1)       │
├─────────────────┼─────────────────┼─────────────────┼─────────────────────┤
│ ┌─────────────┐ │ ┌─────────────┐ │ ┌─────────────┐ │ ┌─────────────────┐ │
│ │payment-refac│ │ │auth-oauth   │ │ │hotfix-login │ │ │ui-redesign-nav  │ │
│ │   codex     │ │ │   claude    │ │ │   claude    │ │ │                 │ │
│ │  td-c3d4    │ │ │  td-a1b2    │ │ │  td-e5f6    │ │ │  td-g7h8        │ │
│ │ +156 -34    │ │ │ +47 -12     │ │ │ +8 -2       │ │ │ +0 -0           │ │
│ └─────────────┘ │ │             │ │ └─────────────┘ │ └─────────────────┘ │
│ ┌─────────────┐ │ │⚡ oauth_call│ │ ┌─────────────┐ │                     │
│ │api-rate-lim │ │ │  back.go    │ │ │readme-update│ │                     │
│ │   claude    │ │ └─────────────┘ │ │   claude    │ │                     │
│ │  td-k9m2    │ │                 │ │  td-p4q5    │ │                     │
│ │ +23 -5      │ │                 │ │ +24 -8      │ │                     │
│ └─────────────┘ │                 │ └─────────────┘ │                     │
│                 │                 │                 │                     │
├─────────────────┴─────────────────┴─────────────────┴─────────────────────┤
│ ←→:columns  j/k:cards  y:approve  ↵:attach  m:merge  v:list view          │
└─────────────────────────────────────────────────────────────────────────────┘
```

**Important:** Columns represent **observed state**, not user-controlled state. Users cannot drag items between columns—status changes based on agent behavior.

### 4.3 New Worktree Modal

Triggered by `n` key:

```
┌────────────────────────────────────────────────────────────────┐
│  New Worktree                                                  │
├────────────────────────────────────────────────────────────────┤
│                                                                │
│  Branch name                                                   │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │ feature/user-preferences                                 │  │
│  └──────────────────────────────────────────────────────────┘  │
│                                                                │
│  Link to TD task (optional)                                    │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │ user pref                                                │  │
│  └──────────────────────────────────────────────────────────┘  │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │ td-x7y8  Add user preferences page                      ◀│  │
│  │ td-z9a1  User preference sync                            │  │
│  └──────────────────────────────────────────────────────────┘  │
│                                                                │
│  Base branch                                                   │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │ main                                                     │  │
│  └──────────────────────────────────────────────────────────┘  │
│                                                                │
│  Agent                                                         │
│  ┌────────┐ ┌────────┐ ┌────────┐ ┌────────┐                   │
│  │◉ Claude│ │○ Codex │ │○ Aider │ │○ None  │                   │
│  └────────┘ └────────┘ └────────┘ └────────┘                   │
│                                                                │
│  Options                                                       │
│  ☑ Copy .env from main worktree                                │
│  ☑ Run setup script (.worktree-setup.sh)                       │
│  ☐ Symlink node_modules                                        │
│                                                                │
├────────────────────────────────────────────────────────────────┤
│                              Cancel (Esc)    Create (Enter)    │
└────────────────────────────────────────────────────────────────┘
```

### 4.4 Keyboard Shortcuts

#### Global

| Key           | Action          |
| ------------- | --------------- |
| `q`, `Ctrl+c` | Quit sidecar    |
| `Tab`         | Next plugin     |
| `Shift+Tab`   | Previous plugin |
| `1-9`         | Jump to plugin  |
| `?`           | Toggle help     |

#### Worktree List

| Key      | Action                                |
| -------- | ------------------------------------- |
| `j`, `↓` | Next worktree                         |
| `k`, `↑` | Previous worktree                     |
| `n`      | New worktree                          |
| `N`      | New worktree with prompt              |
| `Enter`  | Attach to tmux session                |
| `y`      | Approve (send "y" to agent)           |
| `Y`      | Approve all pending                   |
| `Esc`    | Cancel / back                         |
| `d`      | View diff (toggle pane)               |
| `D`      | Delete worktree                       |
| `p`      | Push branch to remote                 |
| `P`      | Create pull request                   |
| `m`      | Merge workflow                        |
| `t`      | Link/unlink td task                   |
| `r`      | Resume paused worktree                |
| `R`      | Refresh all                           |
| `/`      | Filter/search                         |
| `v`      | Toggle list/kanban view               |
| `h`, `l` | Switch left/right pane focus          |
| `Tab`    | Cycle preview tabs (Output/Diff/Task) |

#### When Attached to tmux

| Key      | Action                       |
| -------- | ---------------------------- |
| `Ctrl+q` | Detach and return to sidecar |

---

## 5. Git Worktree Operations

### 5.1 Worktree Location Strategy

By default, worktrees are created as siblings to the main repository:

```
~/code/
├── sidecar/                    # Main repository
│   ├── .git/
│   │   └── worktrees/          # Git's worktree metadata
│   │       ├── auth-oauth-flow/
│   │       └── payment-refactor/
│   └── [source files]
│
└── sidecar-worktrees/          # Actual worktree directories
    ├── auth-oauth-flow/
    │   ├── .git                # File pointing to main .git
    │   ├── .td-root            # Points to ~/code/sidecar
    │   └── [source files]
    │
    └── payment-refactor/
        ├── .git
        ├── .td-root
        └── [source files]
```

**Rationale:**

- Keeps worktrees separate from main repo (avoids accidental commits)
- Easy to find and navigate
- Consistent with Claude Squad's default pattern
- Configurable via `worktreeDir` setting

### 5.2 Creating a Worktree

```go
// CreateWorktree creates a new git worktree
func (m *WorktreeManager) CreateWorktree(opts CreateOptions) (*Worktree, error) {
    // Validate branch name
    if !isValidBranchName(opts.Branch) {
        return nil, fmt.Errorf("invalid branch name: %s", opts.Branch)
    }

    // Check if branch already exists
    branchExists, err := m.branchExists(opts.Branch)
    if err != nil {
        return nil, err
    }

    // Determine worktree path
    worktreePath := filepath.Join(m.worktreeDir, opts.Branch)

    // Build git worktree add command
    args := []string{"worktree", "add"}

    if branchExists {
        // Checkout existing branch
        args = append(args, worktreePath, opts.Branch)
    } else {
        // Create new branch from base
        args = append(args, "-b", opts.Branch, worktreePath)
        if opts.BaseBranch != "" {
            args = append(args, opts.BaseBranch)
        }
    }

    // Execute
    cmd := exec.Command("git", args...)
    cmd.Dir = m.repoRoot
    if output, err := cmd.CombinedOutput(); err != nil {
        return nil, fmt.Errorf("git worktree add failed: %s: %w", output, err)
    }

    // Post-creation setup
    if err := m.setupWorktree(worktreePath, opts); err != nil {
        // Cleanup on failure
        m.removeWorktreeDir(worktreePath)
        return nil, err
    }

    return &Worktree{
        Name:       opts.Branch,
        Path:       worktreePath,
        Branch:     opts.Branch,
        BaseBranch: opts.BaseBranch,
        TaskID:     opts.TaskID,
        Status:     StatusPaused,
        CreatedAt:  time.Now(),
    }, nil
}
```

**Git commands used:**

```bash
# Create new branch and worktree
git worktree add -b feature/auth ../sidecar-worktrees/feature-auth main

# Create worktree for existing branch
git worktree add ../sidecar-worktrees/feature-auth feature/auth

# List worktrees (porcelain format for parsing)
git worktree list --porcelain

# Example porcelain output:
# worktree /Users/dev/code/sidecar
# HEAD abc123def456
# branch refs/heads/main
#
# worktree /Users/dev/code/sidecar-worktrees/feature-auth
# HEAD def456abc123
# branch refs/heads/feature/auth
```

### 5.3 Removing a Worktree

```go
// RemoveWorktree removes a worktree and optionally its branch
func (m *WorktreeManager) RemoveWorktree(name string, opts RemoveOptions) error {
    wt, err := m.GetWorktree(name)
    if err != nil {
        return err
    }

    // Stop agent if running
    if wt.Agent != nil {
        if err := m.StopAgent(wt); err != nil {
            return fmt.Errorf("failed to stop agent: %w", err)
        }
    }

    // Check for uncommitted changes
    if !opts.Force {
        if dirty, err := m.isWorktreeDirty(wt.Path); err != nil {
            return err
        } else if dirty {
            return fmt.Errorf("worktree has uncommitted changes (use --force to override)")
        }
    }

    // Remove worktree
    args := []string{"worktree", "remove"}
    if opts.Force {
        args = append(args, "--force")
    }
    args = append(args, wt.Path)

    cmd := exec.Command("git", args...)
    cmd.Dir = m.repoRoot
    if output, err := cmd.CombinedOutput(); err != nil {
        return fmt.Errorf("git worktree remove failed: %s: %w", output, err)
    }

    // Optionally delete branch
    if opts.DeleteBranch {
        args := []string{"branch", "-d", wt.Branch}
        if opts.Force {
            args[1] = "-D"
        }
        exec.Command("git", args...).Run() // Best effort
    }

    return nil
}
```

**Git commands used:**

```bash
# Remove worktree (fails if dirty)
git worktree remove ../sidecar-worktrees/feature-auth

# Force remove (even if dirty)
git worktree remove --force ../sidecar-worktrees/feature-auth

# Prune stale worktree entries (after manual directory deletion)
git worktree prune

# Delete branch after worktree removal
git branch -d feature/auth    # Safe delete (only if merged)
git branch -D feature/auth    # Force delete
```

### 5.4 Listing Worktrees

```go
// ListWorktrees returns all worktrees with their status
func (m *WorktreeManager) ListWorktrees() ([]*Worktree, error) {
    cmd := exec.Command("git", "worktree", "list", "--porcelain")
    cmd.Dir = m.repoRoot
    output, err := cmd.Output()
    if err != nil {
        return nil, err
    }

    worktrees := parseWorktreeList(output)

    // Enrich with agent status and td task info
    for _, wt := range worktrees {
        // Skip main worktree
        if wt.Path == m.repoRoot {
            continue
        }

        // Check for running agent
        if agent, ok := m.agents[wt.Name]; ok {
            wt.Agent = agent
            wt.Status = agent.DeriveStatus()
        }

        // Load td task link
        wt.TaskID = m.loadTaskLink(wt.Path)

        // Get git stats
        wt.Stats = m.getGitStats(wt.Path)
    }

    return worktrees, nil
}
```

### 5.5 Getting Diff Statistics

```go
// getGitStats returns line change statistics for a worktree
func (m *WorktreeManager) getGitStats(worktreePath string) *GitStats {
    stats := &GitStats{}

    // Get diff stat against base branch
    cmd := exec.Command("git", "diff", "--shortstat", "HEAD")
    cmd.Dir = worktreePath
    output, err := cmd.Output()
    if err == nil {
        // Parse: " 3 files changed, 47 insertions(+), 12 deletions(-)"
        stats.parseShortstat(string(output))
    }

    // Get ahead/behind counts
    cmd = exec.Command("git", "rev-list", "--left-right", "--count", "main...HEAD")
    cmd.Dir = worktreePath
    output, err = cmd.Output()
    if err == nil {
        // Parse: "5\t3" (behind, ahead)
        fmt.Sscanf(string(output), "%d\t%d", &stats.Behind, &stats.Ahead)
    }

    return stats
}
```

**Git commands used:**

```bash
# Get diff statistics
git diff --shortstat HEAD
# Output: 3 files changed, 47 insertions(+), 12 deletions(-)

# Get ahead/behind count relative to main
git rev-list --left-right --count main...HEAD
# Output: 5    3  (5 behind, 3 ahead)

# Get list of changed files
git diff --name-only HEAD
```

### 5.6 Safety: Branch Already Checked Out

Git prevents checking out a branch in multiple worktrees. Handle this case:

```go
func (m *WorktreeManager) CreateWorktree(opts CreateOptions) (*Worktree, error) {
    // ... validation ...

    cmd := exec.Command("git", "worktree", "add", "-b", opts.Branch, worktreePath)
    output, err := cmd.CombinedOutput()

    if err != nil {
        if strings.Contains(string(output), "already checked out") {
            return nil, &BranchCheckedOutError{
                Branch:   opts.Branch,
                Location: extractLocation(string(output)),
            }
        }
        return nil, err
    }

    // ...
}
```

Error message example:

```
fatal: 'feature/auth' is already checked out at '/Users/dev/code/sidecar-worktrees/feature-auth'
```

---

## 6. tmux Integration

### 6.1 Session Naming Convention

tmux sessions created by the worktree manager follow this pattern:

```
sidecar-wt-{worktree-name}
```

Example: `sidecar-wt-auth-oauth-flow`

This allows:

- Easy identification of sidecar-managed sessions
- Cleanup on sidecar exit
- Avoiding conflicts with user sessions

### 6.2 Creating a tmux Session

```go
// StartAgent starts an AI agent in a new tmux session
func (m *WorktreeManager) StartAgent(wt *Worktree, agentType AgentType) error {
    sessionName := fmt.Sprintf("sidecar-wt-%s", sanitizeName(wt.Name))

    // Check if session already exists
    checkCmd := exec.Command("tmux", "has-session", "-t", sessionName)
    if checkCmd.Run() == nil {
        return fmt.Errorf("session %s already exists", sessionName)
    }

    // Get agent command
    agentCmd := m.getAgentCommand(agentType, wt)

    // Create new detached session
    args := []string{
        "new-session",
        "-d",                    // Detached
        "-s", sessionName,       // Session name
        "-c", wt.Path,           // Working directory
    }

    // Optionally set history limit
    // This is done after session creation to avoid tmux version issues

    cmd := exec.Command("tmux", args...)
    if err := cmd.Run(); err != nil {
        return fmt.Errorf("failed to create tmux session: %w", err)
    }

    // Set history limit for scrollback capture
    exec.Command("tmux", "set-option", "-t", sessionName, "history-limit", "10000").Run()

    // Enable mouse mode (optional, for scrolling)
    exec.Command("tmux", "set-option", "-t", sessionName, "mouse", "on").Run()

    // Send the agent command
    sendCmd := exec.Command("tmux", "send-keys", "-t", sessionName, agentCmd, "Enter")
    if err := sendCmd.Run(); err != nil {
        return fmt.Errorf("failed to start agent: %w", err)
    }

    // Create agent record
    agent := &Agent{
        Type:        agentType,
        TmuxSession: sessionName,
        StartedAt:   time.Now(),
        OutputBuf:   ring.New(500), // Last 500 lines
    }

    wt.Agent = agent
    m.agents[wt.Name] = agent

    // Start output polling
    go m.pollAgentOutput(wt)

    return nil
}
```

**tmux commands used:**

```bash
# Create detached session with working directory
tmux new-session -d -s "sidecar-wt-auth-feature" -c "/path/to/worktree"

# Set history limit for scrollback
tmux set-option -t "sidecar-wt-auth-feature" history-limit 10000

# Enable mouse mode
tmux set-option -t "sidecar-wt-auth-feature" mouse on

# Send command to start agent
tmux send-keys -t "sidecar-wt-auth-feature" "claude" Enter

# Check if session exists
tmux has-session -t "sidecar-wt-auth-feature"
```

### 6.3 Capturing Agent Output

This is the core mechanism for displaying live agent output in the TUI:

```go
// pollAgentOutput continuously captures tmux pane content
func (m *WorktreeManager) pollAgentOutput(wt *Worktree) {
    ticker := time.NewTicker(1 * time.Second)
    defer ticker.Stop()

    for {
        select {
        case <-ticker.C:
            if wt.Agent == nil {
                return
            }

            output, err := m.capturePane(wt.Agent.TmuxSession)
            if err != nil {
                // Session may have been killed
                if strings.Contains(err.Error(), "can't find") {
                    wt.Agent = nil
                    wt.Status = StatusPaused
                    return
                }
                continue
            }

            // Update buffer
            wt.Agent.OutputBuf.Update(output)
            wt.Agent.LastOutput = time.Now()

            // Detect status from output
            wt.Status = m.detectStatus(output)
            if wt.Status == StatusWaiting {
                wt.Agent.WaitingFor = m.extractPrompt(output)
            }

        case <-m.stopChan:
            return
        }
    }
}

// capturePane gets the last N lines from a tmux pane
func (m *WorktreeManager) capturePane(sessionName string) (string, error) {
    // -p: Print to stdout (instead of buffer)
    // -S: Start line (-200 = last 200 lines)
    // -t: Target session
    cmd := exec.Command("tmux", "capture-pane", "-p", "-S", "-200", "-t", sessionName)
    output, err := cmd.Output()
    if err != nil {
        return "", fmt.Errorf("capture-pane failed: %w", err)
    }
    return string(output), nil
}
```

**tmux capture-pane options:**

```bash
# Capture last 200 lines, print to stdout
tmux capture-pane -p -S -200 -t "sidecar-wt-auth-feature"

# Capture entire scrollback history
tmux capture-pane -p -S - -t "sidecar-wt-auth-feature"

# Options:
#   -p          Print output to stdout (instead of paste buffer)
#   -S <line>   Starting line number (negative = from end, - = beginning)
#   -E <line>   Ending line number (default: visible bottom)
#   -t <target> Target session:window.pane
#   -e          Include escape sequences (colors)
```

**Reference implementation:** See Claude Squad's [session/tmux.go](https://github.com/smtg-ai/claude-squad/blob/main/session/tmux.go) for production-tested capture logic.

### 6.4 Sending Input to Agents

```go
// SendKeys sends keystrokes to the agent's tmux session
func (m *WorktreeManager) SendKeys(wt *Worktree, keys string) error {
    if wt.Agent == nil {
        return fmt.Errorf("no agent running in worktree %s", wt.Name)
    }

    cmd := exec.Command("tmux", "send-keys", "-t", wt.Agent.TmuxSession, keys)
    return cmd.Run()
}

// Approve sends "y" followed by Enter to approve a prompt
func (m *WorktreeManager) Approve(wt *Worktree) error {
    return m.SendKeys(wt, "y\n")
    // Or: return m.SendKeys(wt, "y", "Enter") with separate args
}

// Reject sends "n" followed by Enter
func (m *WorktreeManager) Reject(wt *Worktree) error {
    return m.SendKeys(wt, "n\n")
}

// SendText sends arbitrary text (e.g., custom prompts)
func (m *WorktreeManager) SendText(wt *Worktree, text string) error {
    // Use -l to send literal text (no key name lookup)
    cmd := exec.Command("tmux", "send-keys", "-l", "-t", wt.Agent.TmuxSession, text)
    if err := cmd.Run(); err != nil {
        return err
    }
    // Send Enter separately
    return exec.Command("tmux", "send-keys", "-t", wt.Agent.TmuxSession, "Enter").Run()
}
```

**tmux send-keys options:**

```bash
# Send "y" and press Enter
tmux send-keys -t "sidecar-wt-auth-feature" "y" Enter

# Send literal text (no key name interpretation)
tmux send-keys -l -t "sidecar-wt-auth-feature" "This is my prompt"

# Special keys: Enter, Escape, Space, Tab, Up, Down, Left, Right
# Ctrl+key: C-c, C-d, C-z
# Alt+key: M-a, M-x

# Send Ctrl+C to interrupt
tmux send-keys -t "sidecar-wt-auth-feature" C-c
```

### 6.5 Attaching to tmux Sessions

When the user presses Enter on a worktree, sidecar should suspend itself and attach to the tmux session:

```go
// AttachToSession suspends sidecar and attaches to the agent's tmux session
func (m *Model) AttachToSession(wt *Worktree) tea.Cmd {
    if wt.Agent == nil {
        return nil
    }

    // Use tea.ExecProcess to suspend Bubble Tea and run tmux attach
    c := exec.Command("tmux", "attach-session", "-t", wt.Agent.TmuxSession)

    return tea.ExecProcess(c, func(err error) tea.Msg {
        return AttachFinishedMsg{Err: err}
    })
}
```

**In BubbleTea, `tea.ExecProcess` handles:**

1. Suspending the TUI
2. Restoring terminal state
3. Running the external command
4. Restoring TUI when command exits

The user can detach from tmux using `Ctrl+b d` (default) or a custom binding like `Ctrl+q`.

**Reference:** See [BubbleTea ExecProcess documentation](https://pkg.go.dev/github.com/charmbracelet/bubbletea#ExecProcess)

### 6.6 Cleanup on Exit

```go
// Cleanup stops all agents and optionally removes tmux sessions
func (m *WorktreeManager) Cleanup(removeSessions bool) error {
    for name, agent := range m.agents {
        // Stop polling
        close(agent.stopChan)

        if removeSessions {
            // Kill tmux session
            exec.Command("tmux", "kill-session", "-t", agent.TmuxSession).Run()
        }

        delete(m.agents, name)
    }
    return nil
}

// CleanupOrphanedSessions removes sidecar-wt-* sessions without running worktrees
func CleanupOrphanedSessions() error {
    cmd := exec.Command("tmux", "list-sessions", "-F", "#{session_name}")
    output, err := cmd.Output()
    if err != nil {
        return nil // No tmux server running
    }

    for _, session := range strings.Split(string(output), "\n") {
        if strings.HasPrefix(session, "sidecar-wt-") {
            // Check if corresponding worktree exists
            // If not, kill the session
            exec.Command("tmux", "kill-session", "-t", session).Run()
        }
    }
    return nil
}
```

**tmux commands used:**

```bash
# List all sessions
tmux list-sessions -F "#{session_name}"

# Kill a specific session
tmux kill-session -t "sidecar-wt-auth-feature"

# Kill all sidecar sessions (bash)
tmux list-sessions -F "#{session_name}" | grep "^sidecar-wt-" | xargs -I{} tmux kill-session -t {}
```

---

## 7. Agent Status Detection

### 7.1 Status Detection Patterns

Detecting whether an agent is waiting for input requires parsing the captured output:

```go
// detectStatus analyzes output to determine agent status
func (m *WorktreeManager) detectStatus(output string) WorktreeStatus {
    lines := strings.Split(output, "\n")

    // Check last ~10 lines for patterns
    checkLines := lines
    if len(lines) > 10 {
        checkLines = lines[len(lines)-10:]
    }
    text := strings.Join(checkLines, "\n")

    // Waiting patterns (agent needs user input)
    waitingPatterns := []string{
        "[Y/n]",           // Claude Code permission prompt
        "[y/N]",           // Alternate capitalization
        "(y/n)",           // Aider style
        "? (Y/n)",         // Interactive prompt
        "Allow edit",      // Claude Code file edit
        "Allow bash",      // Claude Code bash command
        "waiting for",     // Generic waiting
        "Press enter",     // Continue prompt
        "Continue?",
    }

    for _, pattern := range waitingPatterns {
        if strings.Contains(strings.ToLower(text), strings.ToLower(pattern)) {
            return StatusWaiting
        }
    }

    // Done patterns (agent completed)
    donePatterns := []string{
        "Task completed",
        "All done",
        "Finished",
        "exited with code 0",
    }

    for _, pattern := range donePatterns {
        if strings.Contains(text, pattern) {
            return StatusDone
        }
    }

    // Error patterns
    errorPatterns := []string{
        "error:",
        "Error:",
        "failed",
        "exited with code 1",
        "panic:",
    }

    for _, pattern := range errorPatterns {
        if strings.Contains(text, pattern) {
            return StatusError
        }
    }

    // Default: active if recent output
    return StatusActive
}

// extractPrompt pulls out the specific prompt text for display
func (m *WorktreeManager) extractPrompt(output string) string {
    lines := strings.Split(output, "\n")

    // Find line containing prompt
    for i := len(lines) - 1; i >= 0 && i > len(lines)-10; i-- {
        line := lines[i]
        if strings.Contains(line, "[Y/n]") ||
           strings.Contains(line, "Allow edit") ||
           strings.Contains(line, "Allow bash") {
            return strings.TrimSpace(line)
        }
    }
    return ""
}
```

### 7.2 Activity Detection

Determine if an agent is actively working (vs. idle):

```go
// isAgentActive checks if the agent has produced output recently
func (a *Agent) isAgentActive() bool {
    // Consider active if output in last 30 seconds
    return time.Since(a.LastOutput) < 30*time.Second
}

// deriveStatus combines multiple signals
func (a *Agent) DeriveStatus() WorktreeStatus {
    if a.waitingFor != "" {
        return StatusWaiting
    }
    if a.isAgentActive() {
        return StatusActive
    }
    // No recent output but session exists
    return StatusPaused
}
```

### 7.3 Claude Code Hooks (Alternative Approach)

For more reliable status detection with Claude Code, you can use [Claude Code hooks](https://docs.anthropic.com/en/docs/claude-code/hooks-guide):

```json
{
  "hooks": {
    "Notification": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "echo '{\"event\":\"notification\",\"message\":\"$CLAUDE_NOTIFICATION\"}' >> ~/.sidecar/agent-events.jsonl"
          }
        ]
      }
    ],
    "Stop": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "echo '{\"event\":\"stop\",\"timestamp\":\"'$(date -Iseconds)'\"}' >> ~/.sidecar/agent-events.jsonl"
          }
        ]
      }
    ]
  }
}
```

The worktree manager can then watch this file for events:

```go
// watchAgentEvents monitors Claude Code hook output
func (m *WorktreeManager) watchAgentEvents(wt *Worktree) {
    eventFile := filepath.Join(os.Getenv("HOME"), ".sidecar", "agent-events.jsonl")

    // Use fsnotify or tail -f equivalent
    // Parse JSONL and update agent status
}
```

**Reference:** See [Claude Code hooks reference](https://code.claude.com/docs/en/hooks) for full documentation.

---

## 8. TD Task Manager Integration

### 8.1 The Multi-Database Problem

When using worktrees, each worktree needs to access the **same** td database as the main repository. However, td uses a local `.todos/` directory by default.

**Problem:**

```
~/code/sidecar/.todos/db.sqlite          # Main repo's tasks
~/code/sidecar-worktrees/feature/.todos/ # Empty! Different database
```

**Solution: `.td-root` file**

When creating a worktree, write a `.td-root` file that points to the main repo:

```go
// setupTDRoot creates a .td-root file pointing to the main repo
func (m *WorktreeManager) setupTDRoot(worktreePath string) error {
    tdRootPath := filepath.Join(worktreePath, ".td-root")
    return os.WriteFile(tdRootPath, []byte(m.repoRoot), 0644)
}
```

**Required td modification:**

```go
// In td's database resolution logic:
func resolveDBPath() string {
    // Check for .td-root file
    if content, err := os.ReadFile(".td-root"); err == nil {
        rootPath := strings.TrimSpace(string(content))
        return filepath.Join(rootPath, ".todos")
    }

    // Fall back to current directory
    return ".todos"
}
```

This is a ~5-10 line change to td that enables worktree support.

### 8.2 Linking Tasks to Worktrees

```go
// LinkTask associates a td task with a worktree
func (m *WorktreeManager) LinkTask(wt *Worktree, taskID string) error {
    // Validate task exists
    task, err := m.tdClient.GetTask(taskID)
    if err != nil {
        return fmt.Errorf("task not found: %s", taskID)
    }

    // Store link (in worktree metadata)
    linkPath := filepath.Join(wt.Path, ".sidecar-task")
    if err := os.WriteFile(linkPath, []byte(taskID), 0644); err != nil {
        return err
    }

    wt.TaskID = taskID
    return nil
}

// loadTaskLink reads the task link from a worktree
func (m *WorktreeManager) loadTaskLink(worktreePath string) string {
    linkPath := filepath.Join(worktreePath, ".sidecar-task")
    content, err := os.ReadFile(linkPath)
    if err != nil {
        return ""
    }
    return strings.TrimSpace(string(content))
}
```

### 8.3 Auto-Starting Tasks

When creating a worktree with a linked task, automatically start the task in td:

```go
// createWorktreeWithTask creates worktree and starts linked td task
func (m *WorktreeManager) createWorktreeWithTask(opts CreateOptions) (*Worktree, error) {
    // Create worktree
    wt, err := m.CreateWorktree(opts)
    if err != nil {
        return nil, err
    }

    // Start td task
    if opts.TaskID != "" {
        if err := m.tdStartTask(wt.Path, opts.TaskID); err != nil {
            // Log but don't fail
            log.Printf("warning: failed to start td task: %v", err)
        }
    }

    return wt, nil
}

// tdStartTask runs `td start <task-id>` in the worktree
func (m *WorktreeManager) tdStartTask(worktreePath, taskID string) error {
    cmd := exec.Command("td", "start", taskID)
    cmd.Dir = worktreePath
    return cmd.Run()
}
```

### 8.4 Providing Task Context to Agents

When starting an agent, inject task context:

```go
// getAgentCommand builds the command to start an agent with context
func (m *WorktreeManager) getAgentCommand(agentType AgentType, wt *Worktree) string {
    switch agentType {
    case AgentClaude:
        if wt.TaskID != "" {
            // Start claude with task context
            return fmt.Sprintf("claude --prompt \"$(td show %s)\"", wt.TaskID)
        }
        return "claude"

    case AgentCodex:
        return "codex"

    case AgentAider:
        return "aider"

    case AgentGemini:
        return "gemini"

    default:
        return m.config.CustomAgentCommand
    }
}
```

### 8.5 Task Search for UI

For the fuzzy task search in the new worktree modal:

```go
// SearchTasks returns tasks matching a query
func (m *WorktreeManager) SearchTasks(query string) ([]*td.Task, error) {
    // Use td's query language
    cmd := exec.Command("td", "query", fmt.Sprintf("title ~ '%s'", query))
    cmd.Dir = m.repoRoot
    output, err := cmd.Output()
    if err != nil {
        return nil, err
    }

    return parseTDOutput(output)
}

// GetOpenTasks returns all non-closed tasks
func (m *WorktreeManager) GetOpenTasks() ([]*td.Task, error) {
    cmd := exec.Command("td", "list", "--status", "open,in_progress", "--json")
    cmd.Dir = m.repoRoot
    output, err := cmd.Output()
    if err != nil {
        return nil, err
    }

    return parseTDJSON(output)
}
```

---

## 9. Data Persistence

### 9.1 Worktree Metadata

Worktree state is stored in multiple locations:

1. **Git's worktree metadata:** `.git/worktrees/<name>/`
2. **sidecar metadata per worktree:** `<worktree>/.sidecar/`
3. **Project-level sidecar config:** `<repo>/.sidecar/worktrees.json`

```go
// WorktreeMetadata is stored in <worktree>/.sidecar/meta.json
type WorktreeMetadata struct {
    TaskID     string    `json:"taskId,omitempty"`
    AgentType  AgentType `json:"agentType,omitempty"`
    CreatedAt  time.Time `json:"createdAt"`
    CreatedBy  string    `json:"createdBy"` // "sidecar" or "manual"
    BaseBranch string    `json:"baseBranch"`
}

// saveMetadata writes metadata to the worktree
func (m *WorktreeManager) saveMetadata(wt *Worktree) error {
    metaDir := filepath.Join(wt.Path, ".sidecar")
    os.MkdirAll(metaDir, 0755)

    meta := WorktreeMetadata{
        TaskID:     wt.TaskID,
        AgentType:  wt.Agent.Type,
        CreatedAt:  wt.CreatedAt,
        CreatedBy:  "sidecar",
        BaseBranch: wt.BaseBranch,
    }

    data, _ := json.MarshalIndent(meta, "", "  ")
    return os.WriteFile(filepath.Join(metaDir, "meta.json"), data, 0644)
}
```

### 9.2 Runtime State

Runtime state (agents, sessions) is kept in memory and reconstructed on startup:

```go
// Restore reconnects to existing tmux sessions on startup
func (m *WorktreeManager) Restore() error {
    worktrees, err := m.ListWorktrees()
    if err != nil {
        return err
    }

    for _, wt := range worktrees {
        sessionName := fmt.Sprintf("sidecar-wt-%s", sanitizeName(wt.Name))

        // Check if tmux session exists
        cmd := exec.Command("tmux", "has-session", "-t", sessionName)
        if cmd.Run() == nil {
            // Session exists, reconnect
            agent := &Agent{
                TmuxSession: sessionName,
                StartedAt:   time.Now(), // Unknown actual start
            }

            // Load agent type from metadata
            meta := m.loadMetadata(wt.Path)
            if meta != nil {
                agent.Type = meta.AgentType
            }

            wt.Agent = agent
            m.agents[wt.Name] = agent

            // Start polling
            go m.pollAgentOutput(wt)
        }
    }

    return nil
}
```

---

## 10. Configuration

### 10.1 Plugin Configuration

Configuration is part of sidecar's main config file (`~/.config/sidecar/config.json`):

```json
{
  "plugins": {
    "worktree": {
      "enabled": true,
      "refreshInterval": "2s",
      "worktreeDir": "../{project}-worktrees",
      "defaultAgent": "claude",
      "agents": {
        "claude": {
          "command": "claude",
          "promptFlag": "--prompt"
        },
        "codex": {
          "command": "codex"
        },
        "aider": {
          "command": "aider --model anthropic/claude-3-5-sonnet-20241022"
        },
        "gemini": {
          "command": "gemini"
        }
      },
      "setup": {
        "copyEnv": true,
        "runSetupScript": true,
        "setupScriptName": ".worktree-setup.sh",
        "symlinkDirs": []
      },
      "tmux": {
        "historyLimit": 10000,
        "mouseEnabled": true,
        "sessionPrefix": "sidecar-wt-"
      },
      "td": {
        "autoStart": true,
        "autoHandoff": false
      }
    }
  }
}
```

### 10.2 Project-Level Configuration

Projects can override settings via `.sidecar/config.json`:

```json
{
  "worktree": {
    "worktreeDir": "./worktrees",
    "setup": {
      "symlinkDirs": ["node_modules", ".venv"]
    }
  }
}
```

### 10.3 Setup Script

Projects can define `.worktree-setup.sh` to run after worktree creation:

```bash
#!/bin/bash
# .worktree-setup.sh - Runs in new worktree after creation

# Install dependencies
if [ -f "package.json" ]; then
    npm install
fi

if [ -f "requirements.txt" ]; then
    pip install -r requirements.txt
fi

# Copy environment files from main worktree
if [ -n "$MAIN_WORKTREE" ] && [ -f "$MAIN_WORKTREE/.env" ]; then
    cp "$MAIN_WORKTREE/.env" .env
fi

# Start dev server in background (optional)
# npm run dev &
```

The setup script receives these environment variables:

- `MAIN_WORKTREE`: Path to main repository
- `WORKTREE_BRANCH`: Name of the new branch
- `WORKTREE_PATH`: Path to the new worktree

---

## 11. Safety Precautions

### 11.1 Preventing Data Loss

1. **Uncommitted changes check:**

   ```go
   func (m *WorktreeManager) isWorktreeDirty(path string) (bool, error) {
       cmd := exec.Command("git", "status", "--porcelain")
       cmd.Dir = path
       output, err := cmd.Output()
       if err != nil {
           return false, err
       }
       return len(strings.TrimSpace(string(output))) > 0, nil
   }
   ```

2. **Confirmation dialogs for destructive actions:**
   - Deleting worktrees with uncommitted changes
   - Force-removing branches
   - Stopping agents with pending work

3. **Auto-commit on agent completion:**
   ```go
   func (m *WorktreeManager) autoCommitIfConfigured(wt *Worktree) error {
       if !m.config.AutoCommit {
           return nil
       }

       if dirty, _ := m.isWorktreeDirty(wt.Path); dirty {
           cmd := exec.Command("git", "add", "-A")
           cmd.Dir = wt.Path
           cmd.Run()

           msg := fmt.Sprintf("WIP: %s [sidecar auto-commit]", wt.Branch)
           cmd = exec.Command("git", "commit", "-m", msg)
           cmd.Dir = wt.Path
           return cmd.Run()
       }
       return nil
   }
   ```

### 11.2 Preventing Branch Conflicts

1. **Check for existing branches before creation:**

   ```go
   func (m *WorktreeManager) branchExists(name string) (bool, error) {
       cmd := exec.Command("git", "rev-parse", "--verify", name)
       cmd.Dir = m.repoRoot
       return cmd.Run() == nil, nil
   }
   ```

2. **Detect same-file modifications across worktrees:**
   ```go
   func (m *WorktreeManager) detectConflicts() []Conflict {
       var conflicts []Conflict

       // Get modified files in each worktree
       filesByWorktree := make(map[string][]string)
       for _, wt := range m.worktrees {
           files, _ := m.getModifiedFiles(wt.Path)
           filesByWorktree[wt.Name] = files
       }

       // Find overlaps
       for i, wt1 := range m.worktrees {
           for _, wt2 := range m.worktrees[i+1:] {
               overlap := intersection(filesByWorktree[wt1.Name], filesByWorktree[wt2.Name])
               if len(overlap) > 0 {
                   conflicts = append(conflicts, Conflict{
                       Worktrees: []string{wt1.Name, wt2.Name},
                       Files:     overlap,
                   })
               }
           }
       }

       return conflicts
   }
   ```

### 11.3 tmux Session Safety

1. **Unique session names:**

   ```go
   func sanitizeName(name string) string {
       // tmux session names can't contain periods or colons
       name = strings.ReplaceAll(name, ".", "-")
       name = strings.ReplaceAll(name, ":", "-")
       name = strings.ReplaceAll(name, "/", "-")
       return name
   }
   ```

2. **Session existence check:**

   ```go
   func sessionExists(name string) bool {
       cmd := exec.Command("tmux", "has-session", "-t", name)
       return cmd.Run() == nil
   }
   ```

3. **Graceful agent termination:**
   ```go
   func (m *WorktreeManager) StopAgent(wt *Worktree) error {
       if wt.Agent == nil {
           return nil
       }

       // Try graceful interrupt first
       exec.Command("tmux", "send-keys", "-t", wt.Agent.TmuxSession, "C-c").Run()

       // Wait briefly
       time.Sleep(2 * time.Second)

       // Check if still running
       if sessionExists(wt.Agent.TmuxSession) {
           // Force kill
           exec.Command("tmux", "kill-session", "-t", wt.Agent.TmuxSession).Run()
       }

       wt.Agent = nil
       delete(m.agents, wt.Name)
       return nil
   }
   ```

### 11.4 Worktree Cleanup

Handle orphaned worktrees (directory deleted manually):

```go
func (m *WorktreeManager) Prune() error {
    // Let git clean up its metadata
    cmd := exec.Command("git", "worktree", "prune")
    cmd.Dir = m.repoRoot
    return cmd.Run()
}

func (m *WorktreeManager) RepairWorktree(path string) error {
    cmd := exec.Command("git", "worktree", "repair", path)
    cmd.Dir = m.repoRoot
    return cmd.Run()
}
```

---

## 12. Implementation Phases

### Phase 1: Core Infrastructure (MVP)

**Goal:** Basic worktree management without agents

**Features:**

- [ ] Plugin structure and registration
- [ ] List view UI with worktree list
- [ ] Create worktree (branch name, base branch)
- [ ] Delete worktree
- [ ] View diff in preview pane
- [ ] Git stats (additions, deletions)
- [ ] Push to remote

**Estimated effort:** 1-2 weeks

### Phase 2: Agent Integration

**Goal:** Run and monitor agents in worktrees

**Features:**

- [ ] Start agent in tmux session
- [ ] Capture and display agent output
- [ ] Status detection (active/waiting/done)
- [ ] Approve/reject prompts from UI
- [ ] Attach to tmux session
- [ ] Resume on existing worktrees

**Estimated effort:** 2-3 weeks

### Phase 3: TD Integration

**Goal:** Link worktrees to tasks

**Features:**

- [ ] `.td-root` file for worktrees
- [ ] Link task to worktree
- [ ] Fuzzy task search in create modal
- [ ] Auto-start td task
- [ ] Display task info in preview pane

**Estimated effort:** 1 week

**Note:** Requires small modification to td

### Phase 4: Workflow Polish

**Goal:** Streamlined end-to-end experience

**Features:**

- [ ] Merge workflow (diff review → push → PR → merge → cleanup)
- [ ] Setup script support
- [ ] Environment file copying
- [ ] node_modules symlinking
- [ ] Conflict detection
- [ ] Activity timeline/log

**Estimated effort:** 1-2 weeks

### Phase 5: Advanced Features

**Goal:** Power user features

**Features:**

- [ ] Kanban view
- [ ] Batch operations
- [ ] Multiple agent types
- [ ] Claude Code hooks integration
- [ ] Keyboard customization
- [ ] Project-level config

**Estimated effort:** 2-3 weeks

---

## 13. Reference Implementations

### 13.1 Claude Squad

**Repository:** https://github.com/smtg-ai/claude-squad

**Key files to study:**

- `session/tmux.go` - tmux session management, capture-pane
- `session/git/worktree.go` - worktree creation and management
- `ui/` - BubbleTea UI implementation
- `config/` - Configuration handling

**Patterns to adopt:**

- Session naming convention
- Output capture polling
- Auto-yes mode implementation
- Diff view rendering

### 13.2 Treehouse Worktree

**Repository:** https://github.com/mark-hingston/treehouse-worktree

**Key features to study:**

- Lock system for agent coordination
- Setup script execution
- Cleanup with retention policies
- MCP server integration

### 13.3 Sidecar (Existing Plugins)

**Repository:** (this project)

**Key files to study:**

- `internal/plugins/git/` - Git status plugin (diff rendering)
- `internal/plugins/td/` - TD plugin (td integration patterns)
- `internal/tui/` - Shared TUI components

---

## Appendix: Command Reference

### Git Worktree Commands

```bash
# List worktrees
git worktree list
git worktree list --porcelain        # Machine-readable

# Add worktree
git worktree add <path>              # Auto-create branch from path name
git worktree add <path> <branch>     # Checkout existing branch
git worktree add -b <branch> <path>  # Create new branch
git worktree add -b <branch> <path> <base>  # New branch from base

# Remove worktree
git worktree remove <path>           # Fails if dirty
git worktree remove --force <path>   # Force remove

# Maintenance
git worktree prune                   # Clean up stale entries
git worktree prune --dry-run         # Preview what would be pruned
git worktree repair <path>           # Fix broken links

# Locking (prevent prune)
git worktree lock <path>
git worktree lock <path> --reason "Working on feature"
git worktree unlock <path>
```

### tmux Commands

```bash
# Sessions
tmux new-session -d -s <name>        # Create detached
tmux new-session -d -s <name> -c <dir>  # With working directory
tmux kill-session -t <name>          # Kill session
tmux has-session -t <name>           # Check if exists (exit code)
tmux list-sessions                   # List all sessions
tmux list-sessions -F "#{session_name}"  # Names only

# Capturing output
tmux capture-pane -t <session> -p    # Print to stdout
tmux capture-pane -t <session> -p -S -200  # Last 200 lines
tmux capture-pane -t <session> -p -S -     # Entire history
tmux capture-pane -t <session> -p -e       # Include escape sequences

# Sending input
tmux send-keys -t <session> "text" Enter   # Send text + Enter
tmux send-keys -t <session> -l "literal"   # Send without key lookup
tmux send-keys -t <session> C-c            # Send Ctrl+C
tmux send-keys -t <session> Escape         # Send Escape

# Attaching
tmux attach-session -t <name>        # Attach to session
tmux detach-client                   # Detach (from within)

# Options
tmux set-option -t <session> history-limit 10000
tmux set-option -t <session> mouse on
```

### td Commands

```bash
# Tasks
td list                              # List all tasks
td list --status open                # Filter by status
td list --json                       # JSON output
td show <task-id>                    # Task details
td create "Title" --type feature     # Create task
td start <task-id>                   # Start working on task
td handoff <task-id> --done "..." --remaining "..."

# Query
td query "status = open AND priority <= P1"
td query "title ~ 'auth'"

# Session
td usage                             # Current context
td session --new "name"              # New session
```

---

## Document History

| Version     | Date     | Changes               |
| ----------- | -------- | --------------------- |
| 0.1.0-draft | Jan 2026 | Initial specification |

---

_This document is a living specification. Update it as implementation reveals new requirements or constraints._
