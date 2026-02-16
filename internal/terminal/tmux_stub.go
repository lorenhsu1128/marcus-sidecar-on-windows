//go:build windows

package terminal

import "fmt"

// TmuxManager is a stub for Windows platforms.
type TmuxManager struct{}

// NewTmuxManager returns a stub manager on Windows.
func NewTmuxManager() *TmuxManager {
	return &TmuxManager{}
}

func (m *TmuxManager) IsAvailable() bool                      { return false }
func (m *TmuxManager) InstallInstructions() string            { return "tmux is not available on Windows." }
func (m *TmuxManager) CreateSession(name, workDir, cmd string, args []string) (Session, error) {
	return nil, fmt.Errorf("tmux not supported on Windows")
}
func (m *TmuxManager) GetSession(name string) Session         { return nil }
func (m *TmuxManager) HasSession(name string) bool            { return false }
func (m *TmuxManager) ListSessions(prefix string) ([]string, error) { return nil, nil }
func (m *TmuxManager) KillSession(name string) error          { return nil }
func (m *TmuxManager) SetHistoryLimit(name string, lines int) error { return nil }
func (m *TmuxManager) GetPaneID(name string) string                          { return "" }
func (m *TmuxManager) QueryPaneSize(name string) (width, height int, ok bool) { return 0, 0, false }
