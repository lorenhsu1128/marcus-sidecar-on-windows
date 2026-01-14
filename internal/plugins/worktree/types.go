package worktree

import (
	"sync"
	"strings"
	"time"
)

// ViewMode represents the current view state.
type ViewMode int

const (
	ViewModeList   ViewMode = iota // List view (default)
	ViewModeKanban                 // Kanban board view
	ViewModeCreate                 // New worktree modal
)

// FocusPane represents which pane is active in the split view.
type FocusPane int

const (
	PaneSidebar FocusPane = iota // Worktree list
	PanePreview                  // Preview pane (output/diff/task)
)

// PreviewTab represents the active tab in the preview pane.
type PreviewTab int

const (
	PreviewTabOutput PreviewTab = iota // Agent output
	PreviewTabDiff                     // Git diff
	PreviewTabTask                     // TD task info
)

// WorktreeStatus represents the current state of a worktree.
type WorktreeStatus int

const (
	StatusPaused  WorktreeStatus = iota // No agent, worktree exists
	StatusActive                        // Agent running, recent output
	StatusWaiting                       // Agent waiting for input
	StatusDone                          // Agent completed task
	StatusError                         // Agent crashed or errored
)

// String returns the display string for a WorktreeStatus.
func (s WorktreeStatus) String() string {
	switch s {
	case StatusPaused:
		return "paused"
	case StatusActive:
		return "active"
	case StatusWaiting:
		return "waiting"
	case StatusDone:
		return "done"
	case StatusError:
		return "error"
	default:
		return "unknown"
	}
}

// Icon returns the status indicator icon for display.
func (s WorktreeStatus) Icon() string {
	switch s {
	case StatusPaused:
		return "⏸"
	case StatusActive:
		return "●"
	case StatusWaiting:
		return "💬"
	case StatusDone:
		return "✓"
	case StatusError:
		return "✗"
	default:
		return "?"
	}
}

// AgentType represents the type of AI coding agent.
type AgentType string

const (
	AgentClaude AgentType = "claude"
	AgentCodex  AgentType = "codex"
	AgentAider  AgentType = "aider"
	AgentGemini AgentType = "gemini"
	AgentCustom AgentType = "custom"
)

// Worktree represents a git worktree with optional agent.
type Worktree struct {
	Name       string         // e.g., "auth-oauth-flow"
	Path       string         // Absolute path
	Branch     string         // Git branch name
	BaseBranch string         // Branch worktree was created from
	TaskID     string         // Linked td task (e.g., "td-a1b2")
	Agent      *Agent         // nil if no agent running
	Status     WorktreeStatus // Derived from agent state
	Stats      *GitStats      // +/- line counts
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// Agent represents an AI coding agent process.
type Agent struct {
	Type        AgentType     // claude, codex, aider, gemini
	TmuxSession string        // tmux session name
	TmuxPane    string        // Pane identifier
	PID         int           // Process ID (if available)
	StartedAt   time.Time
	LastOutput  time.Time     // Last time output was detected
	OutputBuf   *OutputBuffer // Last N lines of output
	Status      AgentStatus
	WaitingFor  string // Prompt text if waiting
}

// AgentStatus represents the current status of an agent.
type AgentStatus int

const (
	AgentStatusIdle AgentStatus = iota
	AgentStatusRunning
	AgentStatusWaiting
	AgentStatusDone
	AgentStatusError
)

// GitStats holds file change statistics.
type GitStats struct {
	Additions    int
	Deletions    int
	FilesChanged int
	Ahead        int // Commits ahead of base branch
	Behind       int // Commits behind base branch
}

// OutputBuffer is a thread-safe bounded buffer for agent output.
type OutputBuffer struct {
	mu    sync.Mutex
	lines []string
	cap   int
}

// NewOutputBuffer creates a new output buffer with the given capacity.
func NewOutputBuffer(capacity int) *OutputBuffer {
	return &OutputBuffer{
		lines: make([]string, 0, capacity),
		cap:   capacity,
	}
}

// Write appends content to the buffer, trimming old lines if needed.
func (b *OutputBuffer) Write(content string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	newLines := strings.Split(content, "\n")
	b.lines = append(b.lines, newLines...)

	// Trim to capacity
	if len(b.lines) > b.cap {
		b.lines = b.lines[len(b.lines)-b.cap:]
	}
}

// Lines returns a copy of all lines in the buffer.
func (b *OutputBuffer) Lines() []string {
	b.mu.Lock()
	defer b.mu.Unlock()
	result := make([]string, len(b.lines))
	copy(result, b.lines)
	return result
}

// String returns the buffer contents as a single string.
func (b *OutputBuffer) String() string {
	return strings.Join(b.Lines(), "\n")
}

// Clear removes all lines from the buffer.
func (b *OutputBuffer) Clear() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.lines = b.lines[:0]
}

// Len returns the number of lines in the buffer.
func (b *OutputBuffer) Len() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return len(b.lines)
}
