//go:build !windows

package terminal

import "fmt"

// ConPTYManager is a stub for non-Windows platforms.
type ConPTYManager struct{}

// NewConPTYManager returns a stub manager on non-Windows platforms.
func NewConPTYManager() *ConPTYManager {
	return &ConPTYManager{}
}

func (m *ConPTYManager) IsAvailable() bool                      { return false }
func (m *ConPTYManager) InstallInstructions() string            { return "ConPTY is only available on Windows." }
func (m *ConPTYManager) CreateSession(name, workDir, cmd string, args []string) (Session, error) {
	return nil, fmt.Errorf("ConPTY not supported on this platform")
}
func (m *ConPTYManager) GetSession(name string) Session         { return nil }
func (m *ConPTYManager) HasSession(name string) bool            { return false }
func (m *ConPTYManager) ListSessions(prefix string) ([]string, error) { return nil, nil }
func (m *ConPTYManager) KillSession(name string) error          { return nil }
func (m *ConPTYManager) SetHistoryLimit(name string, lines int) error { return nil }
func (m *ConPTYManager) GetPaneID(name string) string                          { return "" }
func (m *ConPTYManager) QueryPaneSize(name string) (width, height int, ok bool) { return 0, 0, false }
