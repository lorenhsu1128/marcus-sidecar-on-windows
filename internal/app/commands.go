package app

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/marcus/sidecar/internal/plugin"
)

// Message types for tea.Cmd
type (
	// TickMsg is sent on each clock tick.
	TickMsg time.Time

	// ToastMsg displays a temporary message.
	ToastMsg struct {
		Message  string
		Duration time.Duration
		IsError  bool // true for error toasts (red), false for success (green)
	}

	// RefreshMsg triggers a full refresh.
	RefreshMsg struct{}

	// ErrorMsg represents an error condition.
	ErrorMsg struct {
		Err error
	}
)

// tickCmd returns a command that ticks every second.
func tickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return TickMsg(t)
	})
}

// ShowToast returns a command to show a toast message.
func ShowToast(msg string, duration time.Duration) tea.Cmd {
	return func() tea.Msg {
		return ToastMsg{
			Message:  msg,
			Duration: duration,
		}
	}
}

// Refresh returns a command to trigger a refresh.
func Refresh() tea.Cmd {
	return func() tea.Msg {
		return RefreshMsg{}
	}
}

// ReportError returns a command to report an error.
func ReportError(err error) tea.Cmd {
	return func() tea.Msg {
		return ErrorMsg{Err: err}
	}
}

// Tick returns a custom tick command with a tag.
func Tick(d time.Duration, tag string) tea.Cmd {
	return tea.Tick(d, func(t time.Time) tea.Msg {
		return TaggedTickMsg{Time: t, Tag: tag}
	})
}

// TaggedTickMsg is a tick with an identifying tag.
type TaggedTickMsg struct {
	Time time.Time
	Tag  string
}

// PluginFocusedMsg is sent to a plugin when it becomes the active plugin.
// Plugins can use this to refresh data or update their state on focus.
// Re-exported from plugin package for backward compatibility.
type PluginFocusedMsg = plugin.PluginFocusedMsg

// PluginFocused returns a command that sends PluginFocusedMsg.
func PluginFocused() tea.Cmd {
	return func() tea.Msg {
		return plugin.PluginFocusedMsg{}
	}
}

// FocusPluginByIDMsg requests focusing a specific plugin by ID.
// Used for cross-plugin navigation (e.g., opening file in file browser from git).
type FocusPluginByIDMsg struct {
	PluginID string
}

// SwitchWorktreeMsg requests switching to a different worktree.
// Used by the worktree switcher modal and workspace plugin "Open in Git Tab" command.
type SwitchWorktreeMsg struct {
	WorktreePath string // Absolute path to the worktree
}

// SwitchWorktree returns a command that requests switching to a worktree by path.
func SwitchWorktree(path string) tea.Cmd {
	return func() tea.Msg {
		return SwitchWorktreeMsg{WorktreePath: path}
	}
}

// WorktreeDeletedMsg is sent when the current worktree has been deleted.
type WorktreeDeletedMsg struct {
	DeletedPath string // Path of the deleted worktree
	MainPath    string // Path to switch to (main worktree)
}

// checkWorktreeExists returns a command that checks if the current worktree still exists.
func checkWorktreeExists(workDir string) tea.Cmd {
	return func() tea.Msg {
		exists, mainPath := CheckCurrentWorktree(workDir)
		if !exists && mainPath != "" {
			return WorktreeDeletedMsg{
				DeletedPath: workDir,
				MainPath:    mainPath,
			}
		}
		return nil
	}
}

// FocusPlugin returns a command that requests focusing a plugin by ID.
func FocusPlugin(pluginID string) tea.Cmd {
	return func() tea.Msg {
		return FocusPluginByIDMsg{PluginID: pluginID}
	}
}

// UpdateSuccessMsg signals that an update completed successfully.
type UpdateSuccessMsg struct {
	SidecarUpdated    bool
	TdUpdated         bool
	NewSidecarVersion string
	NewTdVersion      string
}

// UpdateErrorMsg signals that an update failed.
type UpdateErrorMsg struct {
	Step string // "sidecar", "td", or "check"
	Err  error
}

// UpdateSpinnerTickMsg triggers spinner animation during update.
type UpdateSpinnerTickMsg struct{}

// EditorReturnedMsg signals that an external editor process has exited.
// Used to restore terminal state (mouse support) after returning from vim/etc.
type EditorReturnedMsg struct {
	Err error
}

// updateSpinnerTick returns a command that ticks the spinner every 100ms.
func updateSpinnerTick() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg {
		return UpdateSpinnerTickMsg{}
	})
}

// SwitchToMainWorktreeMsg requests switching to the main worktree.
// Sent when the current WorkDir (a worktree) has been deleted and sidecar
// should gracefully switch to the main repository.
type SwitchToMainWorktreeMsg struct {
	MainWorktreePath string // Path to the main worktree to switch to
}

// SwitchToMainWorktree returns a command that requests switching to the main worktree.
func SwitchToMainWorktree(mainPath string) tea.Cmd {
	return func() tea.Msg {
		return SwitchToMainWorktreeMsg{MainWorktreePath: mainPath}
	}
}
