package terminal

import "runtime"

// NewManager returns the platform-appropriate terminal manager.
// On Unix, returns a tmux-based manager. On Windows, returns a ConPTY-based manager.
func NewManager() Manager {
	if runtime.GOOS == "windows" {
		return NewConPTYManager()
	}
	return NewTmuxManager()
}
