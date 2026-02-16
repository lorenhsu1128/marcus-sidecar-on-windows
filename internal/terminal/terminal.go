// Package terminal defines cross-platform terminal session management interfaces.
// On Unix, the tmux backend is used. On Windows, the ConPTY backend is used.
package terminal

// Session represents a single terminal session (shell or agent).
type Session interface {
	// ID returns the unique session identifier.
	ID() string

	// SendKey sends a named key (e.g. "Enter", "C-c", "Up") to the session.
	SendKey(key string) error

	// SendLiteral sends literal text without interpreting key names.
	SendLiteral(text string) error

	// SendPaste pastes text using the session's buffer mechanism.
	SendPaste(text string) error

	// SendBracketedPaste sends text wrapped in bracketed paste sequences.
	SendBracketedPaste(text string) error

	// SendSGRMouse sends an SGR mouse event.
	// button: 0=left, 1=middle, 2=right. col/row: 1-indexed. release: button release event.
	SendSGRMouse(button, col, row int, release bool) error

	// CaptureOutput captures the current screen content with ANSI codes.
	// scrollback specifies how many lines of history to include.
	CaptureOutput(scrollback int) (string, error)

	// QueryCursor returns cursor position and pane dimensions.
	// row/col are 0-indexed. Returns ok=false if query fails.
	QueryCursor() (row, col, paneHeight, paneWidth int, visible, ok bool)

	// Resize changes the session dimensions.
	Resize(width, height int) error

	// IsAlive returns true if the session process is still running.
	IsAlive() bool

	// Kill terminates the session.
	Kill() error
}

// Manager manages terminal session lifecycle.
type Manager interface {
	// CreateSession creates a new terminal session with the given name and working directory.
	// cmd and args specify the command to run (e.g. "bash" or agent launch command).
	// If cmd is empty, the default shell is used.
	CreateSession(name, workDir, cmd string, args []string) (Session, error)

	// GetSession returns an existing session by name, or nil if not found.
	GetSession(name string) Session

	// HasSession checks if a session with the given name exists and is alive.
	HasSession(name string) bool

	// ListSessions returns names of all sessions matching the prefix.
	ListSessions(prefix string) ([]string, error)

	// KillSession terminates and removes a session by name.
	KillSession(name string) error

	// SetHistoryLimit sets the scrollback buffer size for a session.
	SetHistoryLimit(name string, lines int) error

	// IsAvailable returns true if the terminal backend is available on this platform.
	IsAvailable() bool

	// InstallInstructions returns human-readable install instructions for the backend.
	InstallInstructions() string

	// QueryPaneSize returns the current size of a session's pane.
	QueryPaneSize(name string) (width, height int, ok bool)

	// GetPaneID returns the pane ID for a session. For tmux, this is the pane ID
	// like "%12". For other backends, this may be the session name itself.
	GetPaneID(name string) string
}
